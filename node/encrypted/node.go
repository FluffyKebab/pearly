package encrypted

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"

	"github.com/FluffyKebab/pearly/node"
	"github.com/FluffyKebab/pearly/protocol/id"
	"github.com/FluffyKebab/pearly/transport"
)

const _bitSize = 512

type Node struct {
	node.Node

	idProtoService id.Service
	id             []byte
	privateKey     *rsa.PrivateKey
	publicKey      *rsa.PublicKey
}

var (
	_ node.Node      = &Node{}
	_ node.Encrypted = &Node{}
)

func New(subNode node.Node) (*Node, error) {
	privKey, pubKey, err := generateKeyPair()
	if err != nil {
		return nil, err
	}
	pubKeyBytes := x509.MarshalPKCS1PublicKey(pubKey)

	nodeID := sha256.New()
	nodeID.Write(pubKeyBytes)

	idProtoService := id.Register(subNode, pubKeyBytes, nodeID.Sum(nil))

	return &Node{
		subNode,
		idProtoService,
		nodeID.Sum(nil),
		privKey,
		pubKey,
	}, nil

}

func (n *Node) DialPeerEncrypted(ctx context.Context, ID string) (transport.Conn, error) {
	conn, err := n.DialPeer(ctx, ID)

	//TODO: do id proto to validate peer and get their public key.

	return Conn{
		conn: conn,
	}, err
}

func generateKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	key, err := rsa.GenerateKey(rand.Reader, _bitSize)
	if err != nil {
		return nil, nil, err
	}

	return key, &key.PublicKey, nil
}
