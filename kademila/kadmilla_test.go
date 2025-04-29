package kademila

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/FluffyKebab/pearly/node/basic"
	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/storage"
	"github.com/FluffyKebab/pearly/testutil"
	"github.com/FluffyKebab/pearly/transport/encrypted"
	"github.com/FluffyKebab/pearly/transport/tcp"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/rand"
)

func TestKadmillaUncrypted(t *testing.T) {
	var err error
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*3)
	defer cancelFunc()

	// Creating a network with 4 nodes where everyone knows node 1 and
	// node 1 knows everyone.
	node1, _, addr1 := createUncryptedDHTNode(t, ctx, []byte{0b00000001})
	node2, _, addr2 := createUncryptedDHTNode(t, ctx, []byte{0b00000011})
	node3, _, addr3 := createUncryptedDHTNode(t, ctx, []byte{0b00000111})
	node4, _, addr4 := createUncryptedDHTNode(t, ctx, []byte{0b00001111})

	err = node1.peerstore.AddPeer(peer.New(node2.node.ID(), addr2))
	require.NoError(t, err)
	err = node1.peerstore.AddPeer(peer.New(node3.node.ID(), addr3))
	require.NoError(t, err)
	err = node1.peerstore.AddPeer(peer.New(node4.node.ID(), addr4))
	require.NoError(t, err)

	err = node2.peerstore.AddPeer(peer.New(node1.node.ID(), addr1))
	require.NoError(t, err)
	err = node3.peerstore.AddPeer(peer.New(node1.node.ID(), addr1))
	require.NoError(t, err)
	err = node4.peerstore.AddPeer(peer.New(node1.node.ID(), addr1))
	require.NoError(t, err)

	// Setting a value from node 2 with a key that is closest to 1.
	key1 := []byte{0b00000000}
	value1 := []byte{0b00100001}
	node2.MaxNumStores = 1
	node2.MinNumStores = 1
	err = node2.SetValue(ctx, key1, value1)
	require.NoError(t, err)

	// Checking that node 1 is the only node that stores the value localy.
	_, err = node2.datastore.Get(key1)
	require.ErrorIs(t, err, storage.ErrNotFound)
	_, err = node3.datastore.Get(key1)
	require.ErrorIs(t, err, storage.ErrNotFound)
	_, err = node4.datastore.Get(key1)
	require.ErrorIs(t, err, storage.ErrNotFound)
	valueInNode1, err := node1.datastore.Get(key1)
	require.NoError(t, err)
	require.True(t, bytes.Equal(value1, valueInNode1))

	/// Checking that all nodes are abel to get the value.
	valueGotten1, err := node1.GetValue(ctx, key1)
	require.NoError(t, err)
	require.True(t, bytes.Equal(valueGotten1, value1))

	valueGotten1, err = node2.GetValue(ctx, key1)
	require.NoError(t, err)
	require.True(t, bytes.Equal(valueGotten1, value1))

	valueGotten1, err = node3.GetValue(ctx, key1)
	require.NoError(t, err)
	require.True(t, bytes.Equal(valueGotten1, value1))

	valueGotten1, err = node4.GetValue(ctx, key1)
	require.NoError(t, err)
	require.True(t, bytes.Equal(valueGotten1, value1))
}

func TestKadmilla(t *testing.T) {
	ctx := context.Background()
	numNodes := 20
	numValuesStored := 30

	nodes := make([]DHT, 0, numNodes)
	for i := 0; i < numNodes; i++ {
		newNode, errChan := createEncryptedDHTNode(t, ctx)
		go func() {
			err := <-errChan
			require.NoError(t, err)
		}()

		for i := 0; i < min(3, len(nodes)); i++ {
			previouslyAddedNode := nodes[rand.Intn(len(nodes))]
			err := newNode.Bootstrap(ctx, peer.New(
				nil,
				previouslyAddedNode.node.Transport().ListenAddr(),
			))
			require.NoError(t, err)
		}

		nodes = append(nodes, newNode)
	}

	values := make([][]byte, 0, numValuesStored)
	keys := make([][]byte, 0, numValuesStored)
	for i := 0; i < numValuesStored; i++ {
		curValue := generateRandomString(20)
		values = append(values, []byte(curValue))

		hash, err := storage.NewHasher().Hash([]byte(curValue))
		require.NoError(t, err)
		keys = append(keys, hash)
	}

	for i := 0; i < numValuesStored; i++ {
		randomNode := nodes[rand.Intn(len(nodes))]
		err := randomNode.SetValue(ctx, keys[i], values[i])
		require.NoError(t, err)
	}

	for i := 0; i < numValuesStored; i++ {
		randomNode := nodes[rand.Intn(len(nodes))]
		value, err := randomNode.GetValue(ctx, keys[i])
		require.NoError(t, err)
		require.True(t, bytes.Equal(value, values[i]))
	}
}

func TestBootsrap(t *testing.T) {
	var err error
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*3)
	defer cancelFunc()

	n1, _ := createEncryptedDHTNode(t, ctx)
	n2, _ := createEncryptedDHTNode(t, ctx)

	err = n1.Bootstrap(ctx, peer.New(n2.node.ID(), n2.node.Transport().ListenAddr()))
	require.NoError(t, err)

	// Check that both nodes know eachother.
	peers, _, err := n1.peerstore.GetClosestPeers(n2.node.ID(), 1)
	require.NoError(t, err)
	require.Len(t, peers, 1)
	require.Equal(t, peers[0].ID(), n2.node.ID())

	peers, _, err = n2.peerstore.GetClosestPeers(n1.node.ID(), 1)
	require.NoError(t, err)
	require.Len(t, peers, 1)
	require.Equal(t, peers[0].ID(), n1.node.ID())
}

func TestBootsrapWithoutNodeID(t *testing.T) {
	var err error
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*3)
	defer cancelFunc()

	n1, _ := createEncryptedDHTNode(t, ctx)
	n2, _ := createEncryptedDHTNode(t, ctx)

	err = n1.Bootstrap(ctx, peer.New(nil, n2.node.Transport().ListenAddr()))
	require.NoError(t, err)

	// Check that both nodes know eachother.
	peers, _, err := n1.peerstore.GetClosestPeers(n2.node.ID(), 1)
	require.NoError(t, err)
	require.Len(t, peers, 1)
	require.Equal(t, peers[0].ID(), n2.node.ID())

	peers, _, err = n2.peerstore.GetClosestPeers(n1.node.ID(), 1)
	require.NoError(t, err)
	require.Len(t, peers, 1)
	require.Equal(t, peers[0].ID(), n1.node.ID())
}

func createEncryptedDHTNode(t *testing.T, ctx context.Context) (DHT, <-chan error) {
	t.Helper()

	port, err := testutil.GetAvailablePort()
	require.NoError(t, err)

	transport, err := encrypted.NewTransport(tcp.New(port))
	require.NoError(t, err)

	n := basic.New(transport, transport.ID())
	errChan, err := n.Run(ctx)
	require.NoError(t, err)

	return New(n), errChan
}

func createUncryptedDHTNode(t *testing.T, ctx context.Context, id []byte) (DHT, <-chan error, string) {
	t.Helper()

	port, err := testutil.GetAvailablePort()
	require.NoError(t, err)

	n := basic.New(tcp.New(port), id)
	errChan, err := n.Run(ctx)
	require.NoError(t, err)
	return New(n), errChan, "localhost:" + port
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_-+=<>?~"
	result := make([]rune, 0, length)

	//rand.Seed(uint64(time.Now().UnixNano()))
	for i := 0; i < length; i++ {
		randomRune := rune(charset[rand.Intn(len(charset))])
		result = append(result, randomRune)
	}

	return string(result)
}
