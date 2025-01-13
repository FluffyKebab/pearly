package kdmstore

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"math/big"
	"testing"
	"time"

	"github.com/FluffyKebab/pearly/node/basic"
	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/storage"
	"github.com/FluffyKebab/pearly/testutil"
	"github.com/FluffyKebab/pearly/transport/tcp"
	"github.com/stretchr/testify/require"
)

func TestKDMStoreNoEncryption(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancelFunc()

	client, _, errChan1 := createServiceNoEncryption(t, ctx)
	server, serverData, errChan2 := createServiceNoEncryption(t, ctx)

	req := Request{Key: []byte("key"), Value: []byte("valuevalue")}
	err := client.Do(ctx, req, serverData)
	require.NoError(t, err)

	value, err := server.storer.Get(req.Key)
	require.NoError(t, err)
	require.Equal(t, req.Value, value)

	select {
	case err := <-testutil.CombineErrChan(errChan1, errChan2):
		require.NoError(t, err)
	default:
	}
}

func createServiceNoEncryption(t *testing.T, ctx context.Context) (Service, peer.Peer, <-chan error) {
	t.Helper()
	port, err := testutil.GetAvilablePort()
	require.NoError(t, err)

	transport := tcp.New(port)
	nodeID := makeRandomPeerID(t)
	n := basic.New(transport)
	hashtable := storage.NewHashtable()

	errChan, err := n.Run(ctx)
	require.NoError(t, err)

	service := Register(n, hashtable)
	return service, peer.New(
			nodeID,
			"localhost:"+port),
		testutil.CombineErrChan(errChan, service.Run())
}

func makeRandomPeerID(t *testing.T) []byte {
	t.Helper()

	random, err := rand.Int(rand.Reader, big.NewInt(10000))
	require.NoError(t, err)

	nodeID := sha256.New()
	nodeID.Write(random.Bytes())
	return nodeID.Sum(nil)
}
