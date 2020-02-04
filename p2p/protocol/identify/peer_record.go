package identify

import (
	"context"
	"fmt"
	"github.com/libp2p/go-eventbus"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/record"
	"github.com/multiformats/go-multiaddr"
	"sync"
)

// peerRecordManager creates new signed peer.PeerRecords that can
// be shared with other peers to inform them of our listen addresses in
// a secure and authenticated way.
//
// New signed records are created in response to EvtLocalAddressesUpdated events,
// and are emitted in EvtLocalPeerRecordUpdated events.
//
// EvtLocalPeerRecordUpdated is emitted using a Stateful emitter, so new subscribers
// will immediately receive the current record when they subscribe, with future
// records delivered in future events.
type peerRecordManager struct {
	lock       sync.RWMutex
	latest     *record.Envelope
	hostID     peer.ID
	signingKey crypto.PrivKey

	ctx           context.Context
	subscriptions struct {
		localAddrsUpdated event.Subscription
	}
	emitters struct {
		evtLocalPeerRecordUpdated event.Emitter
	}
}

// NewPeerRecordManager creates a peerRecordManager that will subscribe to the given event.Bus
// and listen for changes in the local Host's addresses, emitting new signed peer.PeerRecords
// in response. The new records will be contained in event.EvtLocalPeerRecordUpdated events
// and emitted on the event bus.
//
// PeerRecords will be signed with the given private key, which must be the libp2p Host's
// identity key for the resulting records to be valid.
//
// If initialAddrs is non-empty, a PeerRecord will be created immediately and emitted on
// the bus, without waiting for an event.LocalPeerAddressesUpdated event to trigger an
// update.
func NewPeerRecordManager(ctx context.Context, bus event.Bus, hostKey crypto.PrivKey, initialAddrs []multiaddr.Multiaddr) (*peerRecordManager, error) {
	hostID, err := peer.IDFromPrivateKey(hostKey)
	if err != nil {
		return nil, err
	}

	m := &peerRecordManager{
		ctx:        ctx,
		signingKey: hostKey,
		hostID:     hostID,
	}

	if len(initialAddrs) != 0 {
		initialRec, err := m.makeSignedPeerRecord(initialAddrs)
		if err != nil {
			return nil, fmt.Errorf("error constructing initial peer record: %w", err)
		}
		m.latest = initialRec
	}

	m.subscriptions.localAddrsUpdated, err = bus.Subscribe(&event.EvtLocalAddressesUpdated{}, eventbus.BufSize(128))
	if err != nil {
		return nil, err
	}
	m.emitters.evtLocalPeerRecordUpdated, err = bus.Emitter(&event.EvtLocalPeerRecordUpdated{}, eventbus.Stateful)
	if err != nil {
		return nil, err
	}

	if m.latest != nil {
		m.emitLatest()
	}

	go m.handleEvents()

	return m, nil
}

// LatestRecord returns the most recently constructed signed PeerRecord.
// If you have a direct reference to the peerRecordManager, this method
// gives you a simple way to access the latest record without pulling
// from the event bus.
func (m *peerRecordManager) LatestRecord() *record.Envelope {
	if m == nil {
		return nil
	}
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.latest
}

func (m *peerRecordManager) handleEvents() {
	sub := m.subscriptions.localAddrsUpdated
	defer func() {
		_ = sub.Close()
		// drain the channel.
		for range sub.Out() {
		}
	}()

	for {
		select {
		case evt, more := <-sub.Out():
			if !more {
				return
			}
			m.update(evt.(event.EvtLocalAddressesUpdated))
		case <-m.ctx.Done():
			return
		}
	}
}

func (m *peerRecordManager) update(evt event.EvtLocalAddressesUpdated) {
	envelope, err := m.makeSignedPeerRecord(addrsFromEvent(evt))
	if err != nil {
		log.Warnf("error creating signed peer record: %v", err)
		return
	}
	m.lock.Lock()
	m.latest = envelope
	m.lock.Unlock()
	m.emitLatest()
}

func (m *peerRecordManager) emitLatest() {
	m.lock.RLock()
	stateEvt := event.EvtLocalPeerRecordUpdated{Record: m.latest}
	m.lock.RUnlock()
	err := m.emitters.evtLocalPeerRecordUpdated.Emit(stateEvt)
	if err != nil {
		log.Warnf("error emitting event for updated peer record: %v", err)
	}
}

func (m *peerRecordManager) makeSignedPeerRecord(current []multiaddr.Multiaddr) (*record.Envelope, error) {
	addrs := make([]multiaddr.Multiaddr, 0, len(current))
	for _, a := range current {
		if a == nil {
			continue
		}
		addrs = append(addrs, a)
	}

	rec := peer.NewPeerRecord()
	rec.PeerID = m.hostID
	rec.Addrs = addrs
	return record.Seal(rec, m.signingKey)
}

func addrsFromEvent(evt event.EvtLocalAddressesUpdated) []multiaddr.Multiaddr {
	addrs := make([]multiaddr.Multiaddr, len(evt.Current))
	for _, a := range evt.Current {
		addrs = append(addrs, a.Address)
	}
	return addrs
}
