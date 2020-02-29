package basichost

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/event"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"

	swarmt "github.com/libp2p/go-libp2p-swarm/testing"
	"github.com/stretchr/testify/require"
)

func TestBasicHostNotifiee(t *testing.T) {
	ctx := context.Background()
	h1 := New(swarmt.GenSwarm(t, ctx))
	defer h1.Close()
	h2 := New(swarmt.GenSwarm(t, ctx))
	defer h2.Close()

	// subscribe for notifications on h1
	s, err := h1.EventBus().Subscribe(&event.EvtPeerStateChange{})
	defer s.Close()
	require.NoError(t, err)

	// connect to h2 so we get the first notificaion
	require.NoError(t, h1.Connect(ctx, peer.AddrInfo{h2.ID(), h2.Addrs()}))
	select {
	case e := <-s.Out():
		evt, ok := e.(event.EvtPeerStateChange)
		require.True(t, ok)
		require.Equal(t, network.Connected, evt.NewState)
		require.Equal(t, h2.ID(), evt.Connection.RemotePeer())
	case <-time.After(1 * time.Second):
		t.Fatal("did not get notification")
	}

	// connect again and make sure we do not get a notification
	require.NoError(t, h2.Connect(ctx, peer.AddrInfo{h1.ID(), h1.Addrs()}))
	select {
	case <-s.Out():
		t.Fatal("should not receive any event")
	case <-time.After(1 * time.Second):
	}

	// disconnect so we get a notification
	require.NoError(t, h1.Network().ClosePeer(h2.ID()))
	select {
	case e := <-s.Out():
		evt, ok := e.(event.EvtPeerStateChange)
		require.True(t, ok)
		require.Equal(t, network.NotConnected, evt.NewState)
		require.Equal(t, h2.ID(), evt.Connection.RemotePeer())
	case <-time.After(1 * time.Second):
		t.Fatal("did not get disconnect notification")
	}
}
