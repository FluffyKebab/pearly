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

type Transport interface {
	Listen(ctx context.Context) (<-chan Conn, <-chan error)
	Dial(context.Context, peer.Peer) (Conn, error)
}
