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

func NewClient(m protocolmux.Muxer, transport transport.Transport) Client {
	return Client{
		muxer:     m,
		transport: transport,
	}
}

func (c Client) EstablishCericut(ctx context.Context, peers []peer.Peer) (transport.Conn, int, error) {
	if len(peers) == 0 {
		return nil, 0, errors.New("missing peers to create circut")
	}

	conn, err := c.transport.Dial(ctx, peers[0])
	if err != nil {
		return nil, 0, fmt.Errorf("dialing peer 1: %w", err)
	}

	for i := 0; i < len(peers)-1; i++ {
		err := c.muxer.SelectProtocol(ctx, OnionProtoID, conn)
		if err != nil {
			return nil, i, err
		}

		transformedConn := transform.NewConn(conn)
		if c.EncryptRequest {
			return nil, 0, errors.New("not implemented")
		}

		curSecretKey := crypto.NewSymetricEncryptionSecretKey()
		err = sendRequest(transformedConn, Request{
			SecretKey:         curSecretKey,
			NextNodeAddr:      peers[i+1].PublicAddr(),
			MaxRandomWaitTime: 0,
		})
		if err != nil {
			return nil, i, err
		}

		err = readResponse(transformedConn)
		if err != nil {
			return nil, i, err
		}

		upgraded, err := crypto.NewEncryptionStream(curSecretKey, conn)
		if err != nil {
			return nil, i, err
		}
		conn = transport.NewConn(upgraded, upgraded, conn)
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
