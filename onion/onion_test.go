package onion

import (
	"context"
	"crypto/rand"
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
		errChan, _ := n.Run(context.Background())
		go func() {
			err := <-errChan
			require.NoError(t, err)
		}()

		peers = append(peers, peer.New(nil, "127.0.0.1:"+curPort))
	}

	curPort, err := testutil.GetAvilablePort()
	require.NoError(t, err)

	finalNode := basic.New(tcp.New(curPort), nil)
	finalNode.Run(context.Background())
	peers = append(peers, peer.New(nil, "127.0.0.1:"+curPort))

	msgSize := 1024 * 9
	msgToSend := make([]byte, msgSize)
	rand.Read(msgToSend)

	finalNode.SetConnHandler(func(c transport.Conn) error {
		t.Helper()

		buf := make([]byte, msgSize)
		numRead := 0
		for numRead < msgSize {
			n, err := c.Read(buf[numRead:])
			require.NoError(t, err)
			numRead += n
		}
		require.Equal(t, msgToSend, buf[:numRead])

		msgSent := make([]byte, msgSize)
		copy(msgSent, msgToSend)
		_, err = c.Write([]byte(msgSent))
		require.NoError(t, err)
		return nil
	})

	clientPort, err := testutil.GetAvilablePort()
	require.NoError(t, err)
	client := NewClient(multistream.NewMuxer(), tcp.New(clientPort))

	conn, _, err := client.EstablishCericut(context.Background(), peers)
	require.NoError(t, err)

	msgSent := make([]byte, msgSize)
	copy(msgSent, msgToSend)
	_, err = conn.Write([]byte(msgSent))
	require.NoError(t, err)

	buf := make([]byte, msgSize)
	n, err := conn.Read(buf)
	require.NoError(t, err)
	require.Equal(t, msgToSend, buf[:n])
}
