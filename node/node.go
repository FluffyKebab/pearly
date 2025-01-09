package node

import (
	"context"

	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/transport"
)

type Node interface {
	Transport() transport.Transport

	SetConnHandler(handler func(transport.Conn))
	RegisterProtocol(protoID string, handler func(transport.Conn))

	Run(context.Context) (<-chan error, error)
	DialPeer(ctx context.Context, p peer.Peer) (transport.Conn, error)
	DialPeerUsingProcol(ctx context.Context, prtoID string, p peer.Peer) (transport.Conn, error)
	ChangeProtocol(ctx context.Context, protoID string, conn transport.Conn) error
}

type DHTHaver interface {
	SetValue(key []byte, value []byte)
	GetValue(key []byte, value []byte)
}
