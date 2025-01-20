package transport

import (
	"context"
	"io"

	"github.com/FluffyKebab/pearly/peer"
)

type Conn interface {
	io.Reader
	io.Writer
	io.Closer
}

type RemoteAddrHaver interface {
	RemoteAddr() string
}

type RemoteIDHaver interface {
	RemoteID() []byte
}

type Transport interface {
	Dial(context.Context, peer.Peer) (Conn, error)
	Listen(ctx context.Context) (<-chan Conn, <-chan error, error)
	ListenAddr() string
}
