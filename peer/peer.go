package peer

type Peer interface {
	ID() string
	PublicAddr() string
}

type peer struct {
	id         string
	publicAddr string
}

func (p peer) ID() string {
	return p.id
}

func (p peer) PublicAddr() string {
	return p.publicAddr
}

func New(id, addr string) Peer {
	return peer{
		id:         id,
		publicAddr: addr,
	}
}
