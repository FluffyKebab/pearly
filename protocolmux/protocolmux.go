package protocolmux

import (
	"context"

	"github.com/FluffyKebab/pearly/transport"
)

type Muxer interface {
	RegisterProtocol(protoID string, handler func(transport.Conn))
	SelectProtocol(ctx context.Context, protoID string, c transport.Conn) error
	HandleConn(transport.Conn) error
}
