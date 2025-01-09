package encrypted

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/testutil"
	"github.com/FluffyKebab/pearly/transport/tcp"
	"github.com/stretchr/testify/require"
)

func TestEncryptedTCP(t *testing.T) {
	port1, err := testutil.GetAvilablePort()
	require.NoError(t, err)

	port2, err := testutil.GetAvilablePort()
	require.NoError(t, err)

	client, err := NewTransport(tcp.New(port1))
	require.NoError(t, err)

	server, err := NewTransport(tcp.New(port2))
	require.NoError(t, err)

	connChan, errChan, err := server.Listen(context.Background())
	require.NoError(t, err)

	conn, err := client.Dial(context.Background(), peer.New("", "localhost:"+port2))
	require.NoError(t, err)

	msg := "no one will be able to read this exept you server!!"
	conn.Write([]byte(msg))
	err = conn.Close()
	require.NoError(t, err)

	timeoutCtx, cancelCtx := context.WithDeadline(context.Background(), time.Now().Add(3*time.Second))
	defer cancelCtx()

	select {
	case <-timeoutCtx.Done():
		t.Fatalf("timeout")
	case conn := <-connChan:
		buf := new(strings.Builder)
		_, err := io.Copy(buf, conn)
		require.NoError(t, err)
		require.Equal(t, msg, buf.String())
		conn.Close()
	case err := <-errChan:
		require.NoError(t, err)
	}
}
