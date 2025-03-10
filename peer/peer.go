package peer

import (
	"crypto/sha256"
	"errors"
	"math/big"
)

var ErrNoSpaceToStorePeer = errors.New("no space in bucket for peer")

type Peer interface {
	ID() []byte
	PublicKey() []byte
	PublicAddr() string
}

type Store interface {
	AddPeer(Peer) error
	RemovePeer(Peer) error
	Peers() []Peer
	GetClosestPeers(ID []byte, k int) ([]Peer, []*big.Int, error)
	Distance(keyA, keyB []byte) (*big.Int, error)
}

type peer struct {
	id         []byte
	pubKey     []byte
	publicAddr string
}

func (p *peer) ID() []byte {
	return p.id
}

func (p *peer) PublicAddr() string {
	return p.publicAddr
}

func (p *peer) PublicKey() []byte {
	return p.pubKey
}

func New(id []byte, addr string) Peer {
	return &peer{
		id:         id,
		publicAddr: addr,
		pubKey:     make([]byte, 0),
	}
}

func NewWithPublicKey(addr string, pubKey []byte) Peer {
	id := sha256.Sum256(pubKey)
	return &peer{
		id:         id[:],
		publicAddr: addr,
		pubKey:     pubKey,
	}
}
