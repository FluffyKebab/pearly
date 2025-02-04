package proxy

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/FluffyKebab/pearly/node"
	"github.com/FluffyKebab/pearly/node/basic"
	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/protocol/proxy"
	"github.com/FluffyKebab/pearly/protocolmux/multistream"
	"github.com/FluffyKebab/pearly/testutil"
	"github.com/FluffyKebab/pearly/transport"
	"github.com/FluffyKebab/pearly/transport/tcp"
	"github.com/stretchr/testify/require"
)

func createProxyServer(t *testing.T) peer.Peer {
	t.Helper()
	port, err := testutil.GetAvilablePort()
	require.NoError(t, err)

	/* transport, err := encrypted.NewTransport(tcp.New(port))
	require.NoError(t, err) */

	n := basic.New(tcp.New(port), nil)
	_, err = n.Run(context.Background())
	require.NoError(t, err)

	proxy.Register(n).Run()
	return peer.New(nil, "localhost:"+port)
}

func createProxyClient(t *testing.T, proxyServer peer.Peer) node.Node {
	t.Helper()
	port, err := testutil.GetAvilablePort()
	require.NoError(t, err)

	//transport, err := encrypted.NewTransport(tcp.New(port))
	//require.NoError(t, err)

	proxyTransport := New(tcp.New(port), multistream.NewMuxer(), proxyServer)
	n := basic.New(proxyTransport, nil)
	n.Run(context.Background())
	return n
}

func TestProxy(t *testing.T) {
	t.Parallel()

	proxyServer := createProxyServer(t)
	c1 := createProxyClient(t, proxyServer)
	c2 := createProxyClient(t, proxyServer)

	msg := "test of message sent from c1 to c2 via the proxy server"
	done := make(chan struct{})

	c2.SetConnHandler(func(c transport.Conn) error {
		buf := new(strings.Builder)
		_, err := io.Copy(buf, c)
		require.NoError(t, err)
		require.Equal(t, msg, buf.String())
		c.Close()
		done <- struct{}{}
		return nil
	})

	conn, err := c1.DialPeer(context.Background(), peer.New(nil, c2.Transport().ListenAddr()))
	require.NoError(t, err)

	_, err = conn.Write([]byte(msg))
	require.NoError(t, err)

	err = conn.Close()
	require.NoError(t, err)
	<-done
}
