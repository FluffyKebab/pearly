package kadmilla

import (
	"bytes"
	"context"
	"errors"
	"math/big"

	"github.com/FluffyKebab/pearly/node"
	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/peer/dhtpeer"
	"github.com/FluffyKebab/pearly/protocol/kdmgetvalue"
	"github.com/FluffyKebab/pearly/protocol/kdmstore"
	"github.com/FluffyKebab/pearly/storage"
)

type DHT struct {
	node              node.Node
	peerstore         peer.Store
	datastore         storage.Hashtable
	hasher            storage.Hasher
	getValueService   kdmgetvalue.Service
	storeValueService kdmstore.Service
}

var _ node.DHT = DHT{}

func New(node node.Node) DHT {
	peerstore := dhtpeer.NewStore(node.ID(), 5)
	hashtable := storage.NewHashtable()

	getValueService := kdmgetvalue.Register(node, peerstore, hashtable)
	storeValueService := kdmstore.Register(node, hashtable)

	return DHT{
		node:              node,
		peerstore:         peerstore,
		datastore:         hashtable,
		getValueService:   getValueService,
		storeValueService: storeValueService,
	}
}

func (dht DHT) SetValue(ctx context.Context, key []byte, value []byte) error {
	return nil
}

func (dht DHT) GetValue(ctx context.Context, key []byte) ([]byte, error) {
	hashedKey, err := dht.hasher.Hash(key)
	if err != nil {
		return nil, err
	}

	value, err := dht.datastore.Get(hashedKey)
	if err == nil {
		return value, nil
	}
	if !errors.Is(err, storage.ErrNotFound) {
		return nil, err
	}

	closestPeers, distences, err := dht.peerstore.GetClosestPeers(hashedKey, 3)
	if err != nil {
		return nil, err
	}

	nodes := make([]searchNode, 0)
	for i := 0; i < len(closestPeers) && i < len(distences); i++ {
		err = dht.peerstore.AddPeer(closestPeers[i])
		if err != nil {
			return nil, err
		}
		nodes = addSearchNode(nodes, closestPeers[i], distences[i], false)
	}

	for {
		node, err := closestNode(nodes)
		if err != nil {
			return nil, err
		}
		node.searchDone = true

		response, err := dht.getValueService.Do(ctx, kdmgetvalue.Request{
			Key: key,
			K:   3,
		}, node.peer)

		if errors.Is(err, kdmgetvalue.ErrInvalidResponse) ||
			errors.Is(err, kdmgetvalue.ErrUnableToReachPeer) ||
			errors.Is(err, kdmgetvalue.ErrInternalServerError) {

			err = dht.peerstore.RemovePeer(node.peer)
			if err != nil {
				return nil, err
			}
			continue
		}
		if err != nil {
			return nil, err
		}
		if response.Value != nil {
			return response.Value, nil
		}

		for i := 0; i < len(response.ClosestNodes); i++ {
			curPeer := peer.New(response.ClosestNodes[i].ID, response.ClosestNodes[i].PublicAddr)
			err = dht.peerstore.AddPeer(curPeer)
			if err != nil {
				return nil, err
			}

			nodes = addSearchNode(
				nodes,
				curPeer,
				response.ClosestNodes[i].Distance,
				false,
			)
		}

	}
}

type searchNode struct {
	peer       peer.Peer
	distance   *big.Int
	searchDone bool
}

func addSearchNode(nodes []searchNode, peer peer.Peer, dist *big.Int, searchDone bool) []searchNode {
	if peerAllreadyAdded(nodes, peer) {
		return nodes
	}
	nodes = append(nodes, searchNode{
		peer:       peer,
		distance:   dist,
		searchDone: searchDone,
	})
	return nodes
}

func closestNode(nodes []searchNode) (*searchNode, error) {
	var closest *searchNode
	for _, node := range nodes {
		if closest == nil {
			closest = &node
			continue
		}
		if closest.distance.Cmp(node.distance) < 0 {
			closest = &node
		}
	}

	return closest, nil
}

func peerAllreadyAdded(nodes []searchNode, p peer.Peer) bool {
	for _, node := range nodes {
		if bytes.Equal(node.peer.ID(), p.ID()) {
			return true
		}
	}
	return false
}
