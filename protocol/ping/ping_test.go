package ping

import (
	"context"
	"testing"
	"time"

	"github.com/FluffyKebab/pearly/node"
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
	node.Node,
	string,
	func(ctx context.Context, nodeID string) (time.Duration, error),
	<-chan error,
) {
	t.Helper()

	port, err := testutil.GetAvilablePort()
	require.NoError(t, err)

	n := basic.New(tcp.New(port))
	errChan1, err := n.Run(ctx)
	require.NoError(t, err)

	callFunc, errChan2 := Register(n)
	return n, port, callFunc, testutil.CombineErrChan(errChan1, errChan2)
}

func TestPing(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelFunc()

	n1, _, callFunc, errChan1 := createNode(t, ctx)
	_, port2, _, errChan2 := createNode(t, ctx)

	n2ID := "2"
	err := n1.RegisterPeer(peer.New(n2ID, "localhost:"+port2))
	require.NoError(t, err)

	result, err := callFunc(ctx, n2ID)
	require.NoError(t, err)
	t.Log(result)

	select {
	case err := <-testutil.CombineErrChan(errChan1, errChan2):
		require.NoError(t, err)
	default:
	}
}
