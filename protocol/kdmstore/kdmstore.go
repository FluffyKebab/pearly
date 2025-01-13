package kdmstore

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"

	"github.com/FluffyKebab/pearly/node"
	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/storage"
	"github.com/FluffyKebab/pearly/transport"
)

var (
	_responseFailed = []byte("failed")
	_responseOK     = []byte("OK")
)

type Request struct {
	Key   []byte
	Value []byte
}

type Service struct {
	node   node.Node
	storer storage.Hashtable
}

func Register(node node.Node, storer storage.Hashtable) Service {
	return Service{
		node:   node,
		storer: storer,
	}
}

func (s Service) Run() <-chan error {
	errChan := make(chan error)
	s.node.RegisterProtocol("/kdmstore", func(c transport.Conn) {
		var req Request
		err := gob.NewDecoder(c).Decode(&req)
		if err != nil {
			errChan <- fmt.Errorf("kdmstore decoding: %w", err)
			c.Write(_responseFailed)
			return
		}

		err = s.storer.Set(req.Key, req.Value)
		if err != nil {
			errChan <- fmt.Errorf("kdmstore storing value: %w", err)
			c.Write(_responseFailed)
			return
		}

		_, err = c.Write(_responseOK)
		if err != nil {
			errChan <- fmt.Errorf("kdmstore sending response: %w", err)
		}
	})

	return errChan
}

func (s Service) Do(ctx context.Context, req Request, peer peer.Peer) error {
	c, err := s.node.DialPeerUsingProcol(ctx, "/kdmstore", peer)
	if err != nil {
		return err
	}

	err = gob.NewEncoder(c).Encode(req)
	if err != nil {
		return err
	}

	buf := make([]byte, 128)
	n, err := c.Read(buf)
	if err != nil {
		return err
	}
	if !bytes.Equal(buf[:n], _responseOK) {
		return fmt.Errorf("storing value failed")
	}
	return nil
}
