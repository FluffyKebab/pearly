package encrypted

import (
	"crypto/rand"
	"crypto/rsa"

	"github.com/FluffyKebab/pearly/transport"
)

type Conn struct {
	conn transport.Conn

	peerPubKey  *rsa.PublicKey
	nodePrivKey *rsa.PrivateKey
}

var _ transport.Conn = Conn{}

func NewConn(underlayingConn transport.Conn, peerPubKey *rsa.PublicKey, nodePrivateKey *rsa.PrivateKey) Conn {
	return Conn{
		conn:        underlayingConn,
		peerPubKey:  peerPubKey,
		nodePrivKey: nodePrivateKey,
	}
}

func (c Conn) Read(p []byte) (n int, err error) {
	n, err = c.conn.Read(p)
	if err != nil {
		return 0, err
	}

	plaintext, err := c.nodePrivKey.Decrypt(rand.Reader, p[0:n], nil)
	if err != nil {
		return 0, err
	}

	for i := 0; i < n; i++ {
		if i >= len(plaintext) {
			p[i] = 0
			continue
		}

		p[i] = plaintext[i]
	}

	return len(plaintext), nil
}

func (c Conn) Write(p []byte) (n int, err error) {
	ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, c.peerPubKey, p)
	if err != nil {
		return 0, err
	}

	return c.conn.Write(ciphertext)
}

func (c Conn) Close() error {
	return c.conn.Close()
}
