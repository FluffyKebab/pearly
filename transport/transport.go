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

type conn struct {
	io.Reader
	io.Writer
	io.Closer
}

func NewConn(r io.Reader, w io.Writer, c io.Closer) Conn {
	return conn{r, w, c}
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
