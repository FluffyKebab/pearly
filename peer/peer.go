package peer

type Peer interface {
	ID() []byte
	PublicAddr() string
}

type Store interface {
	AddPeer(Peer) error
	GetClosestPeers(ID []byte, k int) []Peer
}

type peer struct {
	id         []byte
	publicAddr string
}

func (p peer) ID() []byte {
	return p.id
}

func (p peer) PublicAddr() string {
	return p.publicAddr
}

func New(id []byte, addr string) Peer {
	return peer{
		id:         id,
		publicAddr: addr,
	}
}
