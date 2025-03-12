package onion

import (
	"bufio"
	"context"
	"encoding/gob"
	"errors"
	"fmt"

	"github.com/FluffyKebab/pearly/crypto"
	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/protocolmux"
	"github.com/FluffyKebab/pearly/transport"
	"github.com/FluffyKebab/pearly/transport/transform"
)

type Client struct {
	EncryptRequest bool
	muxer          protocolmux.Muxer
	transport      transport.Transport
}

func (c Client) EstablishCericut(ctx context.Context, peers []peer.Peer) (transport.Conn, int, error) {
	if len(peers) == 0 {
		return nil, 0, errors.New("missing peers to create circut")
	}

	firstConn, err := c.transport.Dial(ctx, peers[0])
	if err != nil {
		return nil, 0, err
	}
	conn := transform.NewConn(firstConn, transform.NOPTransform, transform.NOPTransform)

	for i := 0; i < len(peers)-1; i++ {
		err := c.muxer.SelectProtocol(ctx, OnionProtoID, conn)
		if err != nil {
			return nil, i, err
		}

		if c.EncryptRequest {
			return nil, 0, errors.New("not implemented")
		}

		curSecretKey := crypto.NewSymetricEncryptionSecretKey()
		err = sendRequest(conn, Request{
			SecretKey:         curSecretKey,
			NextNodeAddr:      peers[i+1].PublicAddr(),
			MaxRandomWaitTime: 0,
		})
		if err != nil {
			return nil, i, err
		}

		err = readResponse(conn)
		if err != nil {
			return nil, i, err
		}

		err = upgradeClientConnection(conn, curSecretKey, i)
		if err != nil {
			return nil, i, err
		}
	}

	return conn, 0, err
}

func sendRequest(c *transform.Conn, req Request) error {
	buf := bufio.NewWriterSize(c, 1024)
	err := gob.NewEncoder(buf).Encode(req)
	if err != nil {
		return err
	}

	return buf.Flush()
}

func readResponse(c *transform.Conn) error {
	buf := make([]byte, 1024)
	n, err := c.Read(buf)
	if err != nil {
		return err
	}

	if !(string(buf[:n]) == _sucssesResponse) {
		return errors.New(string(buf[:n]))
	}
	return nil
}

func upgradeClientConnection(conn *transform.Conn, secretKey []byte, i int) error {
	curEncrypter, err := crypto.NewSymetricEncryption(secretKey)
	if err != nil {
		return err
	}

	conn.Transform = func(b []byte) ([]byte, error) {
		b, err := curEncrypter.Encrypt(b)
		if err != nil {
			return nil, fmt.Errorf("encrypting for %v peer: %w", i+1, err)
		}

		return conn.Transform(b)
	}

	conn.Detransform = func(b []byte) ([]byte, error) {
		b, err := conn.Detransform(b)
		if err != nil {
			return nil, err
		}

		b, err = curEncrypter.Decrypt(b)
		if err != nil {
			return nil, fmt.Errorf("decrypting for %v peer: %w", i+1, err)
		}
		return b, nil
	}

	return nil
}
