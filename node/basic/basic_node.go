package basic

import (
	"context"
	"fmt"

	"github.com/FluffyKebab/pearly/node"
	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/transport"
)

type Node struct {
	transport   transport.Transport
	connHandler func(transport.Conn)
	peers       []peer.Peer
}

var _ node.Node = &Node{}

func New(t transport.Transport) *Node {
	return &Node{
		transport: t,
	}
}

func (n *Node) Run(ctx context.Context) (<-chan error, error) {
	connChan, errChan, err := n.transport.Listen(ctx)
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			select {
			case conn := <-connChan:
				if n.connHandler != nil {
					n.connHandler(conn)
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
		if n.peers[i].ID == ID {
			p = &n.peers[i]
			break
		}
	}

	if p == nil {
		return nil, fmt.Errorf("peer not found")
	}

	return n.transport.Dial(ctx, *p)
}

func (n *Node) Transport() transport.Transport {
	return n.transport
}

func (n *Node) SetConnHandler(handler func(transport.Conn)) {
	n.connHandler = handler
}

func (n *Node) RegisterPeer(p peer.Peer) error {
	n.peers = append(n.peers, p)
	return nil
}
