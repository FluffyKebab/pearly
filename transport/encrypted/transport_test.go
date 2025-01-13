package encrypted

import (
	"context"
	"encoding/gob"
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

	conn, err := client.Dial(context.Background(), peer.New([]byte{}, "localhost:"+port2))
	require.NoError(t, err)

	msg := "no one will be able to read this exept you server!! trying very long message lets see if this breaks how long does it need to do to be not"
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

func TestEncryptedTCPSendAndReciveGOB(t *testing.T) {
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

	conn, err := client.Dial(context.Background(), peer.New([]byte{}, "localhost:"+port2))
	require.NoError(t, err)
	serverConn := <-connChan

	type request struct {
		Field1 string
		Field2 []int
	}
	req := request{
		Field1: "halo",
		Field2: []int{4, 6, 2},
	}

	for i := 0; i < 10; i++ {
		encoder := gob.NewEncoder(conn)
		err = encoder.Encode(req)
		require.NoError(t, err)

		var requestGotten request
		decoder := gob.NewDecoder(serverConn)
		err = decoder.Decode(&requestGotten)
		require.NoError(t, err)
		require.Equal(t, req.Field1, requestGotten.Field1)
		require.Equal(t, req.Field2, requestGotten.Field2)

		select {
		case err := <-errChan:
			require.NoError(t, err)
		default:
		}
	}
	conn.Close()
}
