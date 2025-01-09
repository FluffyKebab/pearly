package basic

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/testutil"
	"github.com/FluffyKebab/pearly/transport"
	"github.com/FluffyKebab/pearly/transport/tcp"
	"github.com/stretchr/testify/require"
)

func TestBesicNode(t *testing.T) {
	port1, err := testutil.GetAvilablePort()
	require.NoError(t, err)
	port2, err := testutil.GetAvilablePort()
	require.NoError(t, err)

	node1 := New(tcp.New(port1))
	node2 := New(tcp.New(port2))

	ctx, cancelCtx := context.WithDeadline(context.Background(), time.Now().Add(3*time.Second))
	defer cancelCtx()

	errChan1, err := node1.Run(ctx)
	require.NoError(t, err)

	errChan2, err := node2.Run(ctx)
	require.NoError(t, err)

	msg := "halo from 1"
	msgRecived := make(chan struct{})

	node2.SetConnHandler(func(c transport.Conn) {
		buf := new(strings.Builder)
		_, err := io.Copy(buf, c)
		require.NoError(t, err)
		require.Equal(t, msg, buf.String())

		msgRecived <- struct{}{}
	})

	conn1, err := node1.DialPeer(ctx, peer.New([]byte("2"), "localhost:"+port2))
	require.NoError(t, err)

	conn1.Write([]byte(msg))
	conn1.Close()

	<-msgRecived

	select {
	case err := <-errChan1:
		require.NoError(t, err)
	case err := <-errChan2:
		require.NoError(t, err)
	default:
	}
}
