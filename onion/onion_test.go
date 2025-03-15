package onion

import (
	"context"
	"testing"

	"github.com/FluffyKebab/pearly/node/basic"
	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/protocolmux/multistream"
	"github.com/FluffyKebab/pearly/testutil"
	"github.com/FluffyKebab/pearly/transport"
	"github.com/FluffyKebab/pearly/transport/tcp"
	"github.com/stretchr/testify/require"
)

func TestOnion(t *testing.T) {
	peers := make([]peer.Peer, 0)
	for i := 0; i < 3; i++ {
		curPort, err := testutil.GetAvilablePort()
		require.NoError(t, err)

		n := basic.New(tcp.New(curPort), nil)
		RegisterService(n).Run()
		n.Run(context.Background())

		peers = append(peers, peer.New(nil, "127.0.0.1:"+curPort))
	}

	curPort, err := testutil.GetAvilablePort()
	require.NoError(t, err)

	finalNode := basic.New(tcp.New(curPort), nil)
	finalNode.Run(context.Background())
	peers = append(peers, peer.New(nil, "127.0.0.1:"+curPort))

	msgToSend := "halo this is the message sent very secretly to you"

	finalNode.SetConnHandler(func(c transport.Conn) error {
		t.Helper()
		buf := make([]byte, 1024)
		n, err := c.Read(buf)
		require.NoError(t, err)
		require.Equal(t, msgToSend, string(buf[:n]))

		_, err = c.Write([]byte(msgToSend))
		require.NoError(t, err)
		return nil
	})

	clientPort, err := testutil.GetAvilablePort()
	require.NoError(t, err)
	client := NewClient(multistream.NewMuxer(), tcp.New(clientPort))

	conn, _, err := client.EstablishCericut(context.Background(), peers)
	require.NoError(t, err)

	_, err = conn.Write([]byte(msgToSend))
	require.NoError(t, err)

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	require.NoError(t, err)
	require.Equal(t, msgToSend, string(buf[:n]))
}
