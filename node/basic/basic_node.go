package basic

import (
	"context"
	"fmt"
	"io"

	"github.com/FluffyKebab/pearly/node"
	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/protocolmux"
	"github.com/FluffyKebab/pearly/protocolmux/multistream"
	"github.com/FluffyKebab/pearly/transport"
)

type Node struct {
	transport     transport.Transport
	connHandler   func(transport.Conn)
	peers         []peer.Peer
	protocolMuxer protocolmux.Muxer
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
				for {
					if n.connHandler != nil {
						n.connHandler(conn)
					} else {
						err := n.protocolMuxer.HandleConn(conn)
						if err != nil {
							errChan <- err
						}
					}

					// break if conn is closed.
					if _, err := conn.Read(make([]byte, 1)); err == io.EOF {
						break
					}
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	return errChan, nil
}

func (n *Node) DialPeer(ctx context.Context, ID string) (transport.Conn, error) {
	var p *peer.Peer
	for i := 0; i < len(n.peers); i++ {
		if n.peers[i].ID() == ID {
			p = &n.peers[i]
			break
		}
	}

	if p == nil {
		return nil, fmt.Errorf("peer not found")
	}

	return n.transport.Dial(ctx, *p)
}

func (n *Node) DialPeerUsingProcol(ctx context.Context, prtoID string, peerID string) (transport.Conn, error) {
	c, err := n.DialPeer(ctx, peerID)
	if err != nil {
		return nil, err
	}

	err = n.protocolMuxer.SelectProtocol(prtoID, c)
	return c, err
}

func (n *Node) ChangeProtocol(_ context.Context, protoID string, conn transport.Conn) error {
	return n.protocolMuxer.SelectProtocol(protoID, conn)
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

func (n *Node) RegisterPeer(p peer.Peer) error {
	n.peers = append(n.peers, p)
	return nil
}
