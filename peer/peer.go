package peer

import (
	"errors"
	"math/big"
)

var ErrNoSpaceToStorePeer = errors.New("no space in bucket for peer")

type Peer interface {
	ID() []byte
	SetID(id []byte) *peer

	PublicAddr() string
	SetPublicAddr(addr string) *peer
}

type Store interface {
	AddPeer(Peer) error
	RemovePeer(Peer) error
	GetClosestPeers(ID []byte, k int) ([]Peer, []*big.Int, error)
	Distance(keyA, keyB []byte) (*big.Int, error)
}

type peer struct {
	id         []byte
	publicAddr string
}

func (p *peer) ID() []byte {
	return p.id
}

func (p *peer) PublicAddr() string {
	return p.publicAddr
}

func (p *peer) SetID(id []byte) *peer {
	p.id = id
	return p
}

func (p *peer) SetPublicAddr(addr string) *peer {
	p.publicAddr = addr
	return p
}

func New(id []byte, addr string) Peer {
	return &peer{
		id:         id,
		publicAddr: addr,
	}
}
