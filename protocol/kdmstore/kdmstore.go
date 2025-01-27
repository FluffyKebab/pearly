package kdmstore

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"

	"github.com/FluffyKebab/pearly/node"
	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/storage"
	"github.com/FluffyKebab/pearly/transport"
)

var (
	ErrUnableToReachPeer = errors.New("unable to reach peer")
	ErrInvalidResponse   = errors.New("invalid response from peer")
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

func (s Service) Run() {
	s.node.RegisterProtocol("/kdmstore", func(c transport.Conn) error {
		var req Request
		err := gob.NewDecoder(c).Decode(&req)
		if err != nil {
			c.Write(_responseFailed)
			return fmt.Errorf("kdmstore decoding: %w", err)
		}

		err = s.storer.Set(req.Key, req.Value)
		if err != nil {
			c.Write(_responseFailed)
			return fmt.Errorf("kdmstore storing value: %w", err)
		}

		_, err = c.Write(_responseOK)
		if err != nil {
			return fmt.Errorf("kdmstore sending response: %w", err)
		}

		return nil
	})
}

func (s Service) Do(ctx context.Context, req Request, peer peer.Peer) error {
	c, err := s.node.DialPeerUsingProcol(ctx, "/kdmstore", peer)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnableToReachPeer, err)
	}
	defer c.Close()

	err = gob.NewEncoder(c).Encode(req)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnableToReachPeer, err)
	}

	buf := make([]byte, 128)
	n, err := c.Read(buf)
	if err != nil {
		return err
	}
	if !bytes.Equal(buf[:n], _responseOK) {
		return ErrInvalidResponse
	}
	return nil
}
