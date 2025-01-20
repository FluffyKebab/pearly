package kdmgetvalue

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/FluffyKebab/pearly/node/basic"
	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/peer/dhtpeer"
	"github.com/FluffyKebab/pearly/storage"
	"github.com/FluffyKebab/pearly/testutil"
	"github.com/FluffyKebab/pearly/transport/encrypted"
	"github.com/FluffyKebab/pearly/transport/tcp"
	"github.com/stretchr/testify/require"
)

func TestKDMGetValue(t *testing.T) {
	for i := 0; i < 50; i++ {
		fmt.Println("DOING TEST ", i)
		ctx, cancelFunc := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancelFunc()

		client, _, errChan1 := createService(t, ctx)
		server, serverData, errChan2 := createService(t, ctx)

		err := server.peerstore.AddPeer(peer.New(makeRandomPeerID(t), "peer1"))
		require.NoError(t, err)
		err = server.peerstore.AddPeer(peer.New(makeRandomPeerID(t), "peer2"))
		require.NoError(t, err)
		err = server.peerstore.AddPeer(peer.New(makeRandomPeerID(t), "peer3"))
		require.NoError(t, err)
		err = server.peerstore.AddPeer(peer.New(makeRandomPeerID(t), "peer4"))
		require.NoError(t, err)

		keyToFind := makeRandomPeerID(t)
		res, err := client.Do(ctx, Request{Key: keyToFind, K: 1}, serverData)
		require.NoError(t, err)
		require.Nil(t, res.Value)
		require.Equal(t, 1, len(res.ClosestNodes))

		select {
		case err := <-testutil.CombineErrChan(errChan1, errChan2):
			require.NoError(t, err)
		default:
		}
	}
}

func TestKDMGetValueNoEncryption(t *testing.T) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancelFunc()

	client, _, errChan1 := createServiceNoEncryption(t, ctx)
	server, serverData, errChan2 := createServiceNoEncryption(t, ctx)

	err := server.peerstore.AddPeer(peer.New(makeRandomPeerID(t), "peer1"))
	require.NoError(t, err)
	err = server.peerstore.AddPeer(peer.New(makeRandomPeerID(t), "peer2"))
	require.NoError(t, err)
	err = server.peerstore.AddPeer(peer.New(makeRandomPeerID(t), "peer3"))
	require.NoError(t, err)
	err = server.peerstore.AddPeer(peer.New(makeRandomPeerID(t), "peer4"))
	require.NoError(t, err)

	keyToFind := makeRandomPeerID(t)
	res, err := client.Do(ctx, Request{Key: keyToFind, K: 1}, serverData)
	require.NoError(t, err)
	require.Nil(t, res.Value)
	require.Equal(t, 1, len(res.ClosestNodes))

	select {
	case err := <-testutil.CombineErrChan(errChan1, errChan2):
		require.NoError(t, err)
	default:
	}
}

func createService(t *testing.T, ctx context.Context) (Service, peer.Peer, <-chan error) {
	t.Helper()
	port, err := testutil.GetAvilablePort()
	require.NoError(t, err)

	transport, err := encrypted.NewTransport(tcp.New(port))
	require.NoError(t, err)

	n := basic.New(transport, transport.ID())
	store := dhtpeer.NewStore(transport.ID(), 4)
	hashtable := storage.NewHashtable()

	errChan, err := n.Run(ctx)
	require.NoError(t, err)

	service := Register(n, store, hashtable)
	service.Run()
	return service, peer.New(transport.ID(), "localhost:"+port), errChan
}

func createServiceNoEncryption(t *testing.T, ctx context.Context) (Service, peer.Peer, <-chan error) {
	t.Helper()
	port, err := testutil.GetAvilablePort()
	require.NoError(t, err)

	transport := tcp.New(port)
	nodeID := makeRandomPeerID(t)
	n := basic.New(transport, nil)
	store := dhtpeer.NewStore(nodeID, 4)
	hashtable := storage.NewHashtable()

	errChan, err := n.Run(ctx)
	require.NoError(t, err)

	service := Register(n, store, hashtable)
	service.Run()
	return service, peer.New(nodeID, "localhost:"+port), errChan
}

func makeRandomPeerID(t *testing.T) []byte {
	t.Helper()

	random, err := rand.Int(rand.Reader, big.NewInt(10000))
	require.NoError(t, err)

	nodeID := sha256.New()
	nodeID.Write(random.Bytes())
	return nodeID.Sum(nil)
}
