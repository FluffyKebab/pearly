package dhtpeer

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"math/bits"

	"github.com/FluffyKebab/pearly/peer"
)

var ErrNoSpaceToStorePeer = errors.New("no space in bucket for peer")

type Store struct {
	// Number of peers stored in each bucket.
	k       int
	nodeID  []byte
	buckets [][]peer.Peer
}

var _ peer.Store = Store{}

func NewStore(nodeID []byte, k int) Store {
	buckets := make([][]peer.Peer, len(nodeID)*8)
	for i := 0; i < len(buckets); i++ {
		buckets[i] = make([]peer.Peer, k)
	}

	return Store{
		k:       k,
		nodeID:  nodeID,
		buckets: buckets,
	}
}

func (s Store) AddPeer(p peer.Peer) error {
	if len(s.nodeID) != len(p.ID()) {
		return fmt.Errorf("diffrent len keys") // TODO: imprv
	}

	bucketPos := numEqualBitsPrefix(s.nodeID, p.ID())
	return insertPeerIntoBucket(s.buckets[bucketPos], p)
}

func (s Store) RemovePeer(p peer.Peer) error {
	if len(s.nodeID) != len(p.ID()) {
		return fmt.Errorf("diffrent len keys") // TODO: imprv
	}

	bucketPos := numEqualBitsPrefix(s.nodeID, p.ID())
	removePeerFromBucket(s.buckets[bucketPos], p)
	return nil
}

func (s Store) GetClosestPeers(key []byte, k int) ([]peer.Peer, error) {
	if len(s.nodeID) != len(key) {
		return nil, fmt.Errorf("diffrent len keys") // TODO: imprv
	}

	type peerDistence struct {
		peer.Peer
		*big.Int
	}

	// TODO: imprv
	closest := make([]peerDistence, k)
	for _, bucket := range s.buckets {
		for _, p := range bucket {
			if p == nil {
				continue
			}

			dis := distenceBetween(p.ID(), key)
			for i := 0; i < len(closest); i++ {
				if closest[i].Peer == nil {
					closest[i] = peerDistence{p, dis}
				}

				if dis.Cmp(closest[i].Int) < 0 {
					closest[i] = peerDistence{p, dis}
				}
			}
		}
	}

	res := make([]peer.Peer, 0, len(closest))
	for _, p := range closest {
		if p.Peer == nil {
			break
		}
		res = append(res, p.Peer)
	}
	return res, nil
}

func numEqualBitsPrefix(a, b []byte) int {
	res := 0
	for i := 0; i < len(b); i++ {
		cur := a[i] ^ b[i]
		res += bits.LeadingZeros8(uint8(cur))

		if cur != 8 {
			break
		}
	}

	return res
}

func distenceBetween(id, key []byte) *big.Int {
	xored := make([]byte, len(key))
	for i := 0; i < len(key); i++ {
		xored[i] = id[i] ^ key[i]
	}

	return big.NewInt(0).SetBytes(xored)
}

func insertPeerIntoBucket(bucket []peer.Peer, toBeInserted peer.Peer) error {
	for i, peer := range bucket {
		if peer == nil {
			bucket[i] = toBeInserted
			return nil
		}

		if bytes.Equal(peer.ID(), toBeInserted.ID()) {
			return nil
		}
	}

	return ErrNoSpaceToStorePeer
}

func removePeerFromBucket(bucket []peer.Peer, toBeRemoved peer.Peer) {
	isRemoved := false
	for i := 0; i < len(bucket); i++ {
		if bucket[i] == nil {
			return
		}
		if !isRemoved {
			if !bytes.Equal(bucket[i].ID(), toBeRemoved.ID()) {
				continue
			}

			bucket[i] = nil
			isRemoved = true
		}

		if i+1 == len(bucket) {
			bucket[i] = nil
			continue
		}
		bucket[i] = bucket[i+1]
	}
}
