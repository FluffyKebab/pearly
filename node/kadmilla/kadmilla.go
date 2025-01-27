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
	"github.com/FluffyKebab/pearly/storage"
)

var (
	ErrAllreadySet   = errors.New("a value with this key is allredy set in the DHT")
	ErrSettingFailed = errors.New("failed to set value")
)

type DHT struct {
	node              node.Node
	peerstore         peer.Store
	datastore         storage.Hashtable
	getValueService   kdmgetvalue.Service
	storeValueService kdmstore.Service
}

var _ node.DHT = DHT{}

func New(node node.Node) DHT {
	peerstore := dhtpeer.NewStore(node.ID(), 5)
	hashtable := storage.NewHashtable()

	getValueService := kdmgetvalue.Register(node, peerstore, hashtable)
	storeValueService := kdmstore.Register(node, hashtable)

	getValueService.Run()
	storeValueService.Run()

	return DHT{
		node:              node,
		peerstore:         peerstore,
		datastore:         hashtable,
		getValueService:   getValueService,
		storeValueService: storeValueService,
	}
}

func (dht DHT) SetValue(ctx context.Context, key []byte, value []byte) error {
	nodes, err := dht.intilizeSearchNodeWithSelf(key)
	if err != nil {
		return err
	}
	peersThatShouldStoreValue := make([]peer.Peer, 0)
	errorCollection := make([]errorPeer, 0)

	for {
		newNodesFound, nodeContacted, valueStored, err := dht.searchOnePeer(ctx, nodes, key, 3)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				break
			}

			if isExpectedKDMError(err) {
				errorCollection = append(errorCollection, errorPeer{err, nodeContacted.peer})
				continue
			}

			return err
		}
		if valueStored != nil {
			return ErrAllreadySet
		}

		numNewNodesAdded := 0
		for i := 0; i < len(newNodesFound); i++ {
			if nodeContacted.distance.Cmp(newNodesFound[i].distance) > 0 {
				nodes = addSearchNode(nodes, newNodesFound[i])
				numNewNodesAdded++
			}
		}

		if numNewNodesAdded == 0 {
			// This node is closer then all of thier peers to the key, therfore it should be a storer.
			peersThatShouldStoreValue = append(peersThatShouldStoreValue, nodeContacted.peer)
		}
	}

	if len(peersThatShouldStoreValue) == 0 {
		return fmt.Errorf(
			"%w: unable to reach enough peers to find storers: [%w]",
			ErrSettingFailed,
			combineErrors(errorCollection),
		)
	}

	for _, nonresponder := range errorCollection {
		err = dht.peerstore.RemovePeer(nonresponder.peer)
		if err != nil {
			return err
		}
	}

	failedStored := make([]error, 0)
	for _, peer := range peersThatShouldStoreValue {
		err := dht.storeValueService.Do(ctx, kdmstore.Request{Key: key, Value: value}, peer)
		if err != nil {
			if !(errors.Is(err, kdmstore.ErrInvalidResponse) || errors.Is(err, kdmstore.ErrUnableToReachPeer)) {
				return err
			}

			failedStored = append(failedStored, err)
		}
	}

	if len(peersThatShouldStoreValue) == len(failedStored) {
		return fmt.Errorf(
			"%w: unable to store value in any of the %v storers found: %v",
			ErrSettingFailed,
			len(peersThatShouldStoreValue),
			failedStored,
		)
	}

	return nil
}

func (dht DHT) GetValue(ctx context.Context, key []byte) (value []byte, err error) {
	nodes, err := dht.intilizeSearchNodeWithSelf(key)
	if err != nil {
		return nil, err
	}
	errorCollection := make([]errorPeer, 0)

	for {
		newNodesFound, nodeContacted, valueStored, err := dht.searchOnePeer(ctx, nodes, key, 10)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				break
			}
			if isExpectedKDMError(err) {
				errorCollection = append(errorCollection, errorPeer{err, nodeContacted.peer})
				continue
			}
			return nil, err
		}
		if valueStored != nil {
			return valueStored, nil
		}

		for i := 0; i < len(newNodesFound); i++ {
			nodes = addSearchNode(nodes, newNodesFound[i])
		}
	}

	return nil, fmt.Errorf(
		"%w: possible errors connacting peers: [%w]",
		storage.ErrNotFound,
		combineErrors(errorCollection),
	)
}

func (dht DHT) Bootstrap(ctx context.Context, peerInNetwork peer.Peer) error {
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
	k int,
) (newNodesFound []searchNode, nodeContacted *searchNode, value []byte, err error) {
	node, err := closestNode(nodes)
	if err != nil {
		return nil, nil, nil, err
	}
	node.searchDone = true

	response, err := dht.getValueService.Do(ctx, kdmgetvalue.Request{
		Key: hashedKey,
		K:   k,
	}, node.peer)

	if err != nil {
		return nil, node, nil, err
	}
	if response.Value != nil {
		return nil, node, response.Value, nil
	}

	// Converting the response to search nodes and trying to add peers to the peerstore.
	newNodesFound = make([]searchNode, 0, len(response.ClosestNodes))
	for i := 0; i < len(response.ClosestNodes); i++ {
		curNode := response.ClosestNodes[i]
		curPeer := peer.New(curNode.ID, curNode.PublicAddr)

		err = dht.peerstore.AddPeer(curPeer)
		if err != nil && !errors.Is(err, peer.ErrNoSpaceToStorePeer) {
			return nil, node, nil, fmt.Errorf(
				"adding peer (%s, %s) to peerstore: %w", curPeer.ID(), curPeer.PublicAddr(), err,
			)
		}

		newNodesFound = append(newNodesFound, searchNode{
			peer:       curPeer,
			distance:   curNode.Distance,
			searchDone: false,
		})
	}

	return newNodesFound, node, nil, nil
}

func (dht DHT) intilizeSearchNodeWithSelf(key []byte) ([]searchNode, error) {
	selfDistance, err := dht.peerstore.Distance(dht.node.ID(), key)
	if err != nil {
		return nil, err
	}

	return []searchNode{
		{
			peer:     peer.New(dht.node.ID(), dht.node.Transport().ListenAddr()),
			distance: selfDistance,
		},
	}, nil
}

type searchNode struct {
	peer       peer.Peer
	distance   *big.Int
	searchDone bool
}

func addSearchNode(nodes []searchNode, n searchNode) []searchNode {
	if peerAllreadyAdded(nodes, n.peer) {
		return nodes
	}
	return append(nodes, n)
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

func isExpectedKDMError(err error) bool {
	return errors.Is(err, kdmgetvalue.ErrInvalidResponse) ||
		errors.Is(err, kdmgetvalue.ErrUnableToReachPeer) ||
		errors.Is(err, kdmgetvalue.ErrInternalServerError)
}

type errorPeer struct {
	err  error
	peer peer.Peer
}

func combineErrors(errs []errorPeer) error {
	combinedErrors := errors.New("")
	for _, err := range errs {
		combinedErrors = fmt.Errorf("%w, %w", combinedErrors, err.err)
	}
	return combinedErrors
}
