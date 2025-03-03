package encrypted

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"

	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/transport"
)

const _bitSize = 512

type Transport struct {
	underlaying transport.Transport

	id         []byte
	privateKey *rsa.PrivateKey
	publicKey  []byte
}

var _ transport.Transport = Transport{}

func NewTransport(underalying transport.Transport) (Transport, error) {
	privKey, pubKey, err := generateKeyPair()
	if err != nil {
		return Transport{}, err
	}
	pubKeyBytes := x509.MarshalPKCS1PublicKey(pubKey)
	nodeID := sha256.Sum256(pubKeyBytes)

	return Transport{
		underlaying: underalying,
		id:          nodeID[:],
		privateKey:  privKey,
		publicKey:   pubKeyBytes,
	}, nil
}

func (t Transport) Listen(ctx context.Context) (<-chan transport.Conn, <-chan error, error) {
	encryptedChanConn := make(chan transport.Conn)
	errChan := make(chan error)

	chanConn, underlayingErrChan, err := t.underlaying.Listen(ctx)
	if err != nil {
		return nil, nil, err
	}

	go func() {
		for {
			select {
			case c := <-chanConn:
				encryptedChan, err := t.upgradeConn(c)
				if err != nil {
					errChan <- err
					continue
				}
				encryptedChanConn <- encryptedChan
			case err := <-underlayingErrChan:
				errChan <- err
			case <-ctx.Done():
				return
			}
		}
	}()

	return encryptedChanConn, errChan, nil
}

func (t Transport) Dial(ctx context.Context, p peer.Peer) (transport.Conn, error) {
	conn, err := t.underlaying.Dial(ctx, p)
	if err != nil {
		return nil, err
	}

	return t.upgradeConn(conn)
}

func (t Transport) ListenAddr() string {
	return t.underlaying.ListenAddr()
}

func (t Transport) ID() []byte {
	return t.id
}

func generateKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	key, err := rsa.GenerateKey(rand.Reader, _bitSize)
	if err != nil {
		return nil, nil, err
	}

	return key, &key.PublicKey, nil
}
