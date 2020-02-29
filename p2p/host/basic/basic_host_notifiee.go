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
	// Is this our first connection with the peer ?
	p := v.RemotePeer()
	if len(n.ConnsToPeer(p)) == 1 {
		evt := event.EvtPeerStateChange{v, network.Connected}
		b.basicHost().emitters.evtPeerStateChange.Emit(evt)
	}
}

func (b *basicHostNotifiee) Disconnected(n network.Network, v network.Conn) {
	// Are we no longer connected to the peer ?
	p := v.RemotePeer()
	if n.Connectedness(p) == network.NotConnected {
		evt := event.EvtPeerStateChange{v, network.NotConnected}
		b.basicHost().emitters.evtPeerStateChange.Emit(evt)
	}
}

func (b *basicHostNotifiee) OpenedStream(n network.Network, v network.Stream) {}

func (b *basicHostNotifiee) ClosedStream(n network.Network, v network.Stream) {}

func (b *basicHostNotifiee) Listen(n network.Network, a ma.Multiaddr) {}

func (b *basicHostNotifiee) ListenClose(n network.Network, a ma.Multiaddr) {}
