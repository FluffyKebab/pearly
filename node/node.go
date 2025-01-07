package node

import (
	"context"

	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/transport"
)

type Node interface {
	Transport() transport.Transport

	SetConnHandler(handler func(transport.Conn))
	RegisterPeer(peer.Peer) error
	RegisterProtocol(protoID string, handler func(transport.Conn))

	Run(context.Context) (<-chan error, error)
	DialPeer(ctx context.Context, ID string) (transport.Conn, error)
	DialPeerUsingProcol(ctx context.Context, prtoID string, peerID string) (transport.Conn, error)
}
