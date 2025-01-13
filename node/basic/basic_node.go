package basic

import (
	"context"

	"github.com/FluffyKebab/pearly/node"
	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/protocolmux"
	"github.com/FluffyKebab/pearly/protocolmux/multistream"
	"github.com/FluffyKebab/pearly/transport"
)

type Node struct {
	transport     transport.Transport
	protocolMuxer protocolmux.Muxer
	connHandler   func(transport.Conn)
}

var _ node.Node = &Node{}

func New(t transport.Transport) *Node {
	return &Node{
		transport:     t,
		protocolMuxer: multistream.NewMuxer(),
	}
}

func (n *Node) Run(ctx context.Context) (<-chan error, error) {
	connChan, transportErrChan, err := n.transport.Listen(ctx)
	if err != nil {
		return nil, err
	}

	errChan := make(chan error)
	go func() {
		for {
			select {
			case err, ok := <-transportErrChan:
				if !ok {
					return
				}
				errChan <- err
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case conn := <-connChan:
				if n.connHandler != nil {
					n.connHandler(conn)
					continue
				}

				err := n.protocolMuxer.HandleConn(conn)
				if err != nil {
					errChan <- err
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	return errChan, nil
}

func (n *Node) DialPeer(ctx context.Context, p peer.Peer) (transport.Conn, error) {
	return n.transport.Dial(ctx, p)
}

func (n *Node) DialPeerUsingProcol(ctx context.Context, prtoID string, p peer.Peer) (transport.Conn, error) {
	c, err := n.DialPeer(ctx, p)
	if err != nil {
		return nil, err
	}

	err = n.protocolMuxer.SelectProtocol(ctx, prtoID, c)
	return c, err
}

func (n *Node) ChangeProtocol(ctx context.Context, protoID string, conn transport.Conn) error {
	return n.protocolMuxer.SelectProtocol(ctx, protoID, conn)
}

func (n *Node) Transport() transport.Transport {
	return n.transport
}

func (n *Node) SetConnHandler(handler func(transport.Conn)) {
	n.connHandler = handler
}

func (n *Node) RegisterProtocol(protoID string, handler func(transport.Conn)) {
	n.protocolMuxer.RegisterProtocol(protoID, handler)
}
