package tcp

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/FluffyKebab/pearly/peer"
	"github.com/stretchr/testify/require"
)

func getAvilablePort() (int, error) {
	a, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	l, err := net.ListenTCP("tcp", a)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func TestTCPTransport(t *testing.T) {
	port1, err := getAvilablePort()
	require.NoError(t, err)

	port2, err := getAvilablePort()
	require.NoError(t, err)

	client := New(strconv.Itoa(port1))
	server := New(strconv.Itoa(port2))

	connChan, errChan := server.Listen(context.Background())
	select {
	case err := <-errChan:
		require.NoError(t, err)
	default:
	}

	conn, err := client.Dial(context.Background(), peer.Peer{PublicAddr: "localhost:" + server.port})
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
		fmt.Println("herjo")
		buf := new(strings.Builder)
		_, err := io.Copy(buf, conn)
		require.NoError(t, err)
		require.Equal(t, msg, buf.String())
	case err := <-errChan:
		require.NoError(t, err)
	}
}
