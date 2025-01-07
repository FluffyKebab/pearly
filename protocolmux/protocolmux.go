package protocolmux

import "github.com/FluffyKebab/pearly/transport"

type Muxer interface {
	RegisterProtocol(protoID string, handler func(transport.Conn))
	SelectProtocol(protoID string, c transport.Conn) error
	HandleConn(transport.Conn) error
}
