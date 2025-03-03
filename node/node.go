package node

import (
	"context"

	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/transport"
)

type Node interface {
	ID() []byte
	Transport() transport.Transport
	Run(context.Context) (<-chan error, error)
	SetConnHandler(handler func(transport.Conn) error)
	RegisterProtocol(protoID string, handler func(transport.Conn) error)
	DialPeer(ctx context.Context, p peer.Peer) (transport.Conn, error)
	DialPeerUsingProcol(ctx context.Context, prtoID string, p peer.Peer) (transport.Conn, error)
}
