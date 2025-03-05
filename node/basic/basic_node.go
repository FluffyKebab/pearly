package basic

import (
	"context"
	"fmt"

	"github.com/FluffyKebab/pearly/node"
	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/protocolmux"
	"github.com/FluffyKebab/pearly/protocolmux/multistream"
	"github.com/FluffyKebab/pearly/transport"
)

type Node struct {
	id            []byte
	transport     transport.Transport
	protocolMuxer protocolmux.Muxer
	connHandler   func(transport.Conn) error
	errChan       chan error
}

var _ node.Node = &Node{}

func New(t transport.Transport, id []byte) *Node {
	return &Node{
		id:            id,
		transport:     t,
		protocolMuxer: multistream.NewMuxer(),
		errChan:       make(chan error),
	}
}

func (n *Node) Run(ctx context.Context) (<-chan error, error) {
	connChan, transportErrChan, err := n.transport.Listen(ctx)
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			select {
			case err, ok := <-transportErrChan:
				if !ok {
					return
				}
				n.errChan <- err
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		for {
			select {
			case conn := <-connChan:
				go n.handleConn(conn)
			case <-ctx.Done():
				return
			}
		}
	}()

	return n.errChan, nil
}

func (n *Node) handleConn(conn transport.Conn) {
	if n.connHandler != nil {
		err := n.connHandler(conn)
		if err != nil {
			n.errChan <- err
		}

		return
	}

	err := n.protocolMuxer.HandleConn(conn)
	if err != nil {
		n.errChan <- err
	}
}

func (n *Node) DialPeer(ctx context.Context, p peer.Peer) (transport.Conn, error) {
	return n.transport.Dial(ctx, p)
}

func (n *Node) DialPeerUsingProcol(ctx context.Context, prtoID string, p peer.Peer) (transport.Conn, error) {
	c, err := n.DialPeer(ctx, p)
	if err != nil {
		fmt.Println("dail peer using proto failed")
		return nil, err
	}

	err = n.protocolMuxer.SelectProtocol(ctx, prtoID, c)
	return c, err
}

func (n *Node) SendError(err error) {
	n.errChan <- err
}

func (n *Node) ID() []byte {
	return n.id
}

func (n *Node) Transport() transport.Transport {
	return n.transport
}

func (n *Node) SetConnHandler(handler func(transport.Conn) error) {
	n.connHandler = handler
}

func (n *Node) RegisterProtocol(protoID string, handler func(transport.Conn) error) {
	n.protocolMuxer.RegisterProtocol(protoID, handler)
}
