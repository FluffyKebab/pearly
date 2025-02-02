package multiuse

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/testutil"
	"github.com/FluffyKebab/pearly/transport/encrypted"
	"github.com/FluffyKebab/pearly/transport/tcp"
	"github.com/stretchr/testify/require"
)

func TestMultiuseWithEncryptedTCP(t *testing.T) {
	t.Parallel()

	t1 := createTransport(t)
	t2 := createTransport(t)

	_, errChan1, err := t1.Listen(context.Background())
	require.NoError(t, err)
	transportChan2, errChan2, err := t2.Listen(context.Background())
	require.NoError(t, err)

	msgSent := "heeeelo from the first created transport"
	numChanOpened := 10
	doneRecived := make(chan struct{})

	go func() {
		for i := 0; i < numChanOpened; i++ {
			conn := <-transportChan2

			buf := new(strings.Builder)
			_, err = io.Copy(buf, conn)
			require.NoError(t, err)
			require.Equal(t, msgSent, buf.String())
		}
		doneRecived <- struct{}{}
	}()

	go func() {
		err := <-testutil.CombineErrChan(errChan1, errChan2)
		require.NoError(t, err)
	}()

	for i := 0; i < numChanOpened; i++ {
		conn, err := t1.Dial(context.Background(), peer.New(nil, t2.ListenAddr()))
		require.NoError(t, err)

		_, err = conn.Write([]byte(msgSent))
		require.NoError(t, err)

		err = conn.Close()
		require.NoError(t, err)
	}

	<-doneRecived
}

func createTransport(t *testing.T) *Transport {
	t.Helper()
	port, err := testutil.GetAvilablePort()
	require.NoError(t, err)

	e, err := encrypted.NewTransport(tcp.New(port))
	require.NoError(t, err)

	return New(e, time.Second*15)
}
