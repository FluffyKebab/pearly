package tcp

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/testutil"
	"github.com/stretchr/testify/require"
)

func TestTCPTransport(t *testing.T) {
	port1, err := testutil.GetAvilablePort()
	require.NoError(t, err)

	port2, err := testutil.GetAvilablePort()
	require.NoError(t, err)

	client := New(port1)
	server := New(port2)

	connChan, errChan, err := server.Listen(context.Background())
	require.NoError(t, err)

	conn, err := client.Dial(context.Background(), peer.New([]byte{}, "localhost:"+server.port))
	require.NoError(t, err)

	msg := "helo"
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
	case err := <-errChan:
		require.NoError(t, err)
	}
}
