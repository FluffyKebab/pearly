package ping

import (
	"context"
	"testing"
	"time"

	"github.com/FluffyKebab/pearly/node/basic"
	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/testutil"
	"github.com/FluffyKebab/pearly/transport/tcp"
	"github.com/stretchr/testify/require"
)

func createNode(
	t *testing.T,
	ctx context.Context,
) (
	string,
	Service,
	<-chan error,
) {
	t.Helper()

	port, err := testutil.GetAvilablePort()
	require.NoError(t, err)

	n := basic.New(tcp.New(port), nil)
	errChan, err := n.Run(ctx)
	require.NoError(t, err)

	pingService := Register(n)
	pingService.Run()

	return port, pingService, errChan
}

func TestPing(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelFunc()

	_, pingService, errChan1 := createNode(t, ctx)
	port2, _, errChan2 := createNode(t, ctx)

	_, err := pingService.Do(ctx, peer.New(nil, "localhost:"+port2))
	require.NoError(t, err)

	select {
	case err := <-testutil.CombineErrChan(errChan1, errChan2):
		require.NoError(t, err)
	default:
	}
}
