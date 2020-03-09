package basichost

import (
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/network"

	ma "github.com/multiformats/go-multiaddr"
)

// basicHostNotifiee listens to Network Notifiee events & emits them on the host's event bus.
// Please read the docs for the "EvtPeerConnectednessChanged" event in go-libp2p-core for an explanation
// of the subtleties involved in getting this right.
type basicHostNotifiee BasicHost

func (b *basicHostNotifiee) basicHost() *BasicHost {
	return (*BasicHost)(b)
}

func (b *basicHostNotifiee) Connected(n network.Network, v network.Conn) {
	p := v.RemotePeer()

	// While the swarm Connected/Disconnected notifications for a single connection are guaranteed
	// to be serialized, notifications for different connections between the same peer are not
	// as they are fired concurrently on different go-routines i.e. We could observe a "Connected" notification
	// for a new connection Conn2 with a peer Px before we observe the "Disconnected" notification for an existing
	// connection Conn1 with Px even though Conn1 was disconnected before Conn2 was created.
	// This striped lock ensures that both "Connected" & "Disconnected" notifications synchronize on
	// "change in connectedness" between two peers.
	indexForLk := len(p) - 1
	lk := &b.stripedConnNotifLocks[p[indexForLk]]
	lk.Lock()
	defer lk.Unlock()

	// what's our last observed connectedness for this peer ?
	b.observedConnectednessLk.RLock()
	prevConn := b.observedConnectedness[p]
	b.observedConnectednessLk.RUnlock()

	// If our last observed connectedness for the peer is "NotConnected" & we are now connected to the peer,
	// we should emit an event.
	if prevConn == network.NotConnected && n.Connectedness(p) == network.Connected {
		// change our observed connectedness for the peer to Connected.
		b.observedConnectednessLk.Lock()
		b.observedConnectedness[p] = network.Connected
		b.observedConnectednessLk.Unlock()

		// emit the event on the bus
		evt := event.EvtPeerConnectednessChanged{p, network.Connected}
		b.basicHost().emitters.evtPeerStateChange.Emit(evt)
	}
}

func (b *basicHostNotifiee) Disconnected(n network.Network, v network.Conn) {
	p := v.RemotePeer()

	// use the last byte of the peer as the key for the striped lock
	indexForLk := len(p) - 1
	lk := &b.stripedConnNotifLocks[p[indexForLk]]
	lk.Lock()
	defer lk.Unlock()

	// what's the last known connectedness for this peer ?
	b.observedConnectednessLk.RLock()
	prevConn := b.observedConnectedness[p]
	b.observedConnectednessLk.RUnlock()

	// If our last observed connectedness for the peer is "Connected" & we now have no connections to the peer,
	// we should emit an event.
	if prevConn == network.Connected && n.Connectedness(p) == network.NotConnected {
		b.observedConnectednessLk.Lock()
		b.observedConnectedness[p] = network.NotConnected
		b.observedConnectednessLk.Unlock()

		evt := event.EvtPeerConnectednessChanged{p, network.NotConnected}
		b.basicHost().emitters.evtPeerStateChange.Emit(evt)
	}
}

func (b *basicHostNotifiee) OpenedStream(n network.Network, v network.Stream) {}

func (b *basicHostNotifiee) ClosedStream(n network.Network, v network.Stream) {}

func (b *basicHostNotifiee) Listen(n network.Network, a ma.Multiaddr) {}

func (b *basicHostNotifiee) ListenClose(n network.Network, a ma.Multiaddr) {}
