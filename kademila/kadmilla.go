package kademila

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/FluffyKebab/pearly/kademila/kdmgetvalue"
	"github.com/FluffyKebab/pearly/kademila/kdmstore"
	"github.com/FluffyKebab/pearly/node"
	"github.com/FluffyKebab/pearly/peer"
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

	NumPeerReturnedSet int
	NumPeerReturnedGet int

	// The maximum number of times a value is replacted accros the newtork
	// when setting a value.
	MaxNumStores int

	// The minimum amount of nodes that need to store a value for a set
	// opporation to be seen as succsesful. Defaults to 2.
	MinNumStores int
}

func New(node node.Node, opts ...Option) DHT {
	option := defualtOptions(node.ID())
	for _, opt := range opts {
		opt(option)
	}

	getValueService := kdmgetvalue.Register(node, option.peerstore, option.datastore)
	storeValueService := kdmstore.Register(node, option.datastore)

	getValueService.Run()
	storeValueService.Run()

	return DHT{
		node:              node,
		peerstore:         option.peerstore,
		datastore:         option.datastore,
		getValueService:   getValueService,
		storeValueService: storeValueService,

		NumPeerReturnedSet: 10,
		NumPeerReturnedGet: 10,
		MaxNumStores:       5,
		MinNumStores:       2,
	}
}

func (dht DHT) SetValue(ctx context.Context, key []byte, value []byte) error {
	nodes, err := dht.intilizeSearchNodeWithSelfForSet(key)
	if err != nil {
		return err
	}

	errorCollection := make([]errorPeer, 0)
	for {
		newNodesFound, nodeContacted, valueStored, err := dht.searchOnePeer(ctx, nodes, key, dht.NumPeerReturnedSet)
		if err != nil {
			// There are no search nodes left that are not searched.
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
		for _, newNode := range newNodesFound {
			for i := 0; i < len(nodes); i++ {
				if nodes[i].peer == nil {
					nodes[i] = newNode
					numNewNodesAdded++
					break
				}

				if bytes.Equal(nodes[i].peer.ID(), newNode.peer.ID()) {
					break
				}

				if newNode.distance.Cmp(nodes[i].distance) < 0 {
					nodes[i] = newNode
					numNewNodesAdded++
					break
				}
			}
		}

		if numNewNodesAdded == 0 {
			break
		}
	}

	resultNodes := make([]searchNode, 0, len(nodes))
	for _, node := range nodes {
		if node.peer == nil {
			break
		}
		resultNodes = append(resultNodes, node)
	}

	if len(resultNodes) < dht.MinNumStores {
		return fmt.Errorf(
			"%w: unable to find minum amount of storers (%v): [%w]",
			ErrSettingFailed,
			dht.MinNumStores,
			combineErrors(errorCollection),
		)
	}

	for _, nonresponder := range errorCollection {
		err = dht.peerstore.RemovePeer(nonresponder.peer)
		if err != nil {
			return err
		}
	}

	failedStored := make([]errorPeer, 0)
	for _, node := range resultNodes {
		err := dht.storeValueService.Do(ctx, kdmstore.Request{Key: key, Value: value}, node.peer)
		if err != nil {
			if !(errors.Is(err, kdmstore.ErrInvalidResponse) || errors.Is(err, kdmstore.ErrUnableToReachPeer)) {
				return err
			}

			failedStored = append(failedStored, errorPeer{err: err})
		}
	}

	if len(resultNodes)-len(failedStored) < dht.MinNumStores {
		return fmt.Errorf(
			"%w: number of succseful duplications of value accros the network (%v) is less then the minium set (%v). %w",
			ErrSettingFailed,
			len(resultNodes)-len(failedStored),
			dht.MinNumStores,
			combineErrors(failedStored),
		)
	}

	return nil
}

func (dht DHT) GetValue(ctx context.Context, key []byte) (value []byte, err error) {
	nodes, err := dht.intilizeSearchNodeWithSelfForGet(key)
	if err != nil {
		return nil, err
	}
	errorCollection := make([]errorPeer, 0)

	for {
		newNodesFound, nodeContacted, valueStored, err := dht.searchOnePeer(ctx, nodes, key, dht.NumPeerReturnedGet)
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

	return dht.peerstore.AddPeer(peer.New(response.NodeContacted.ID, peerInNetwork.PublicAddr()))
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

func (dht DHT) intilizeSearchNodeWithSelfForSet(key []byte) ([]searchNode, error) {
	selfDistance, err := dht.peerstore.Distance(dht.node.ID(), key)
	if err != nil {
		return nil, err
	}

	if dht.MaxNumStores <= 0 {
		return nil, errors.New("MaxNumStores must be larger then 0")
	}

	res := make([]searchNode, dht.MaxNumStores)
	res[0] = searchNode{
		peer:     peer.New(dht.node.ID(), dht.node.Transport().ListenAddr()),
		distance: selfDistance,
	}

	return res, nil
}

func (dht DHT) intilizeSearchNodeWithSelfForGet(key []byte) ([]searchNode, error) {
	selfDistance, err := dht.peerstore.Distance(dht.node.ID(), key)
	if err != nil {
		return nil, err
	}

	return []searchNode{{
		peer:     peer.New(dht.node.ID(), dht.node.Transport().ListenAddr()),
		distance: selfDistance,
	}}, nil
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
		if nodes[i].peer == nil {
			continue
		}
		if nodes[i].searchDone {
			continue
		}

		if closest == nil {
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
		if node.peer == nil {
			return false
		}

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
