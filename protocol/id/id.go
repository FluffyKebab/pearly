package id

import (
	"github.com/FluffyKebab/pearly/node"
	"github.com/FluffyKebab/pearly/transport"
)

type Service struct {
	n         node.Node
	publicKey []byte
	id        []byte
}

func (s Service) Do(conn transport.Conn) ([]byte, []byte, error) {
	return nil, nil, nil
}

func Register(n node.Node, publicKey []byte, nodeID []byte) Service {
	return Service{
		n:         n,
		publicKey: publicKey,
		id:        nodeID,
	}
}
