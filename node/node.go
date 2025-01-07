package node

import (
	"context"

	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/transport"
)

type Node interface {
	Transport() transport.Transport
	SetConnHandler(handler func(transport.Conn))

	Run(context.Context) (<-chan error, error)
	RegisterPeer(peer.Peer) error
	DialPeer(ctx context.Context, ID string) (transport.Conn, error)

	//AddProtocolHandler(protoID string, handler func(transport.Conn))
}
