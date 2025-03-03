package kademila

import (
	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/peer/dhtpeer"
	"github.com/FluffyKebab/pearly/storage"
)

type Option func(*options)

type options struct {
	peerstore          peer.Store
	datastore          storage.Hashtable
	numPeerReturnedSet int
	numPeerReturnedGet int
}

func defualtOptions(nodeID []byte) *options {
	return &options{
		peerstore:          dhtpeer.NewStore(nodeID, 5),
		datastore:          storage.NewHashtable(),
		numPeerReturnedSet: 4,
		numPeerReturnedGet: 10,
	}
}

func WithPeerstore(store peer.Store) Option {
	return func(o *options) {
		o.peerstore = store
	}
}

func WithDatastore(ds storage.Hashtable) Option {
	return func(o *options) {
		o.datastore = ds
	}
}

func WithNumPeerReturnedSet(num int) Option {
	return func(o *options) {
		o.numPeerReturnedSet = num
	}
}

func WithNumPeerReturnedGet(num int) Option {
	return func(o *options) {
		o.numPeerReturnedGet = num
	}
}
