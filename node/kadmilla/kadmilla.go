package kadmilla

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/FluffyKebab/pearly/node"
	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/peer/dhtpeer"
	"github.com/FluffyKebab/pearly/protocol/kdmgetvalue"
	"github.com/FluffyKebab/pearly/protocol/kdmstore"
	"github.com/FluffyKebab/pearly/protocol/ping"
	"github.com/FluffyKebab/pearly/storage"
)

var ErrAllreadySet = errors.New("a value with this key is allredy set in the DHT")

type DHT struct {
	node              node.Node
	peerstore         peer.Store
	datastore         storage.Hashtable
	getValueService   kdmgetvalue.Service
	pingService       ping.Service
	storeValueService kdmstore.Service
}

var _ node.DHT = DHT{}

func New(node node.Node) DHT {
	peerstore := dhtpeer.NewStore(node.ID(), 5)
	hashtable := storage.NewHashtable()

	getValueService := kdmgetvalue.Register(node, peerstore, hashtable)
	storeValueService := kdmstore.Register(node, hashtable)
	pingService := ping.Register(node)

	getValueService.Run()
	storeValueService.Run()
	pingService.Run()

	return DHT{
		node:              node,
		peerstore:         peerstore,
		datastore:         hashtable,
		getValueService:   getValueService,
		pingService:       pingService,
		storeValueService: storeValueService,
	}
}

func (dht DHT) SetValue(ctx context.Context, key []byte, value []byte) error {
	nodes := []searchNode{
		{peer: peer.New(dht.node.ID(), dht.node.Transport().ListenAddr())},
	}
	peersThatShouldStoreValue := make([]peer.Peer, 0)

	for {
		var (
			isClosestToKey bool
			peerSearched   peer.Peer
			valueStored    []byte
			err            error
		)
		nodes, valueStored, isClosestToKey, peerSearched, err = dht.searchOnePeer(ctx, nodes, key)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				break
			}
			return err
		}
		if valueStored != nil {
			return ErrAllreadySet
		}
		if isClosestToKey {
			peersThatShouldStoreValue = append(peersThatShouldStoreValue, peerSearched)
		}
	}

	if len(peersThatShouldStoreValue) == 0 {
		return fmt.Errorf("unexpected error: zero nodes found")
	}

	for _, peer := range peersThatShouldStoreValue {
		err := dht.storeValueService.Do(ctx, kdmstore.Request{Key: key, Value: value}, peer)
		if err != nil { // TODO: handle error.
			return err
		}
	}

	return nil
}

func (dht DHT) GetValue(ctx context.Context, key []byte) (value []byte, err error) {
	nodes := []searchNode{
		{peer: peer.New(dht.node.ID(), dht.node.Transport().ListenAddr())},
	}

	for {
		nodes, value, _, _, err = dht.searchOnePeer(ctx, nodes, key)
		if err != nil {
			return nil, err
		}
		if value != nil {
			return value, err
		}
	}
}

func (dht DHT) Bootstrap(ctx context.Context, peerInNetwork peer.Peer) error {
	// Preform self lookup.
	response, err := dht.getValueService.Do(ctx, kdmgetvalue.Request{
		Key: dht.node.ID(),
		K:   0,
	}, peerInNetwork)
	if err != nil {
		return fmt.Errorf("self lookup failed: %w", err)
	}

	return dht.peerstore.AddPeer(peerInNetwork.SetID(response.NodeContacted.ID))
}

func (dht DHT) searchOnePeer(
	ctx context.Context,
	nodes []searchNode,
	hashedKey []byte,
) ([]searchNode, []byte, bool, peer.Peer, error) {
	node, err := closestNode(nodes)
	if err != nil {
		return nodes, nil, false, nil, err
	}
	node.searchDone = true

	response, err := dht.getValueService.Do(ctx, kdmgetvalue.Request{
		Key: hashedKey,
		K:   3,
	}, node.peer)

	if errors.Is(err, kdmgetvalue.ErrInvalidResponse) ||
		errors.Is(err, kdmgetvalue.ErrUnableToReachPeer) ||
		errors.Is(err, kdmgetvalue.ErrInternalServerError) {

		err = dht.peerstore.RemovePeer(node.peer)
		if err != nil {
			return nodes, nil, false, node.peer, err
		}
		return nodes, nil, false, node.peer, nil
	}
	if err != nil {
		return nodes, nil, false, node.peer, fmt.Errorf("unkown error contatcting peer: %w", err)
	}
	if response.Value != nil {
		return nodes, response.Value, false, node.peer, nil
	}

	for i := 0; i < len(response.ClosestNodes); i++ {
		curPeer := peer.New(response.ClosestNodes[i].ID, response.ClosestNodes[i].PublicAddr)
		err = dht.peerstore.AddPeer(curPeer)
		if err != nil {
			return nodes, nil, false, node.peer, fmt.Errorf(
				"adding peer (%s, %s) to peerstore: %w", curPeer.ID(), curPeer.PublicAddr(), err,
			)
		}

		nodes = addSearchNode(
			nodes,
			curPeer,
			response.ClosestNodes[i].Distance,
			false,
		)
	}

	return nodes, nil, len(response.ClosestNodes) == 0, node.peer, nil
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
	for i := 0; i < len(nodes); i++ {
		if nodes[i].searchDone {
			continue
		}
		if closest == nil || nodes[i].distance == nil {
			closest = &nodes[i]
			continue
		}
		if closest.distance.Cmp(nodes[i].distance) < 0 {
			closest = &nodes[i]
		}
	}

	if closest == nil {
		return nil, storage.ErrNotFound
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
