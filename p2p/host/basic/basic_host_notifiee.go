package basichost

import (
	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/network"

	ma "github.com/multiformats/go-multiaddr"
)

// basicHostNotifiee listens to Network Notifiee events & emits them on the host's event bus
type basicHostNotifiee BasicHost

func (b *basicHostNotifiee) basicHost() *BasicHost {
	return (*BasicHost)(b)
}

func (b *basicHostNotifiee) Connected(n network.Network, v network.Conn) {
	p := v.RemotePeer()

	// use the last byte of the peer as the key for the striped lock
	indexForLk := len(p) - 1
	lk := &b.stripedConnNotifLocks[p[indexForLk]]
	lk.Lock()
	defer lk.Unlock()

	if n.Connectedness(p) == network.Connected {
		evt := event.EvtPeerStateChange{v, network.Connected}
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

	// Are we no longer connected to the peer ?
	if n.Connectedness(p) == network.NotConnected {
		evt := event.EvtPeerStateChange{v, network.NotConnected}
		b.basicHost().emitters.evtPeerStateChange.Emit(evt)
	}
}

func (b *basicHostNotifiee) OpenedStream(n network.Network, v network.Stream) {}

func (b *basicHostNotifiee) ClosedStream(n network.Network, v network.Stream) {}

func (b *basicHostNotifiee) Listen(n network.Network, a ma.Multiaddr) {}

func (b *basicHostNotifiee) ListenClose(n network.Network, a ma.Multiaddr) {}
