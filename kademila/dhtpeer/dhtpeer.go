package dhtpeer

import (
	"bytes"
	"fmt"
	"math/big"
	"math/bits"

	"github.com/FluffyKebab/pearly/peer"
)

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
	if bytes.Equal(p.ID(), s.nodeID) {
		return nil
	}

	bucketPos := numEqualBitsPrefix(s.nodeID, p.ID())
	return insertPeerIntoBucket(s.buckets[bucketPos], p)
}

func (s Store) RemovePeer(p peer.Peer) error {
	if len(s.nodeID) != len(p.ID()) {
		return fmt.Errorf("diffrent len keys") // TODO: imprv
	}
	if bytes.Equal(p.ID(), s.nodeID) {
		return nil
	}

	bucketPos := numEqualBitsPrefix(s.nodeID, p.ID())
	removePeerFromBucket(s.buckets[bucketPos], p)
	return nil
}

func (s Store) Peers() []peer.Peer {
	peers := make([]peer.Peer, 0)
	for i := 0; i < len(s.buckets); i++ {
		for j := 0; j < len(s.buckets[i]); j++ {
			if s.buckets[i][j] != nil {
				peers = append(peers, s.buckets[i][j])
			}
		}
	}

	return peers
}

func (s Store) GetClosestPeers(key []byte, k int) ([]peer.Peer, []*big.Int, error) {
	if len(s.nodeID) != len(key) {
		return nil, nil, fmt.Errorf("diffrent len keys") // TODO: imprv
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
					break
				}

				if dis.Cmp(closest[i].Int) < 0 {
					closest[i] = peerDistence{p, dis}
					break
				}
			}
		}
	}

	res := make([]peer.Peer, 0, len(closest))
	dis := make([]*big.Int, 0, len(closest))
	for _, p := range closest {
		if p.Peer == nil {
			break
		}
		res = append(res, p.Peer)
		dis = append(dis, p.Int)
	}
	return res, dis, nil
}

func (s Store) Distance(keyA, keyB []byte) (*big.Int, error) {
	if len(keyA) != len(keyB) {
		return nil, fmt.Errorf("diffrent len keys in dis") // TODO: imprv
	}

	return distenceBetween(keyA, keyB), nil
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
		if peer != nil {
			if bytes.Equal(peer.ID(), toBeInserted.ID()) {
				return nil
			}
			continue
		}

		bucket[i] = toBeInserted
		return nil
	}

	return peer.ErrNoSpaceToStorePeer
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
