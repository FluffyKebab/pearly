package kdmgetvalue

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"math/big"

	"github.com/FluffyKebab/pearly/node"
	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/storage"
	"github.com/FluffyKebab/pearly/transport"
)

var (
	ErrUnableToReachPeer   = errors.New("unable to reach peer")
	ErrInvalidResponse     = errors.New("invalid response from peer")
	ErrInvalidRequest      = errors.New("invalid request from peer")
	ErrInternalServerError = errors.New("internal server error")
)

type Request struct {
	Key []byte
	K   int
}

type Response struct {
	Value        []byte
	ClosestNodes []Node
	Err          error
}

type Node struct {
	ID         []byte
	Distance   *big.Int
	PublicAddr string
}

type Service struct {
	node      node.Node
	peerstore peer.Store
	storer    storage.Hashtable
}

func Register(node node.Node, peerstore peer.Store, storer storage.Hashtable) Service {
	return Service{
		node:      node,
		peerstore: peerstore,
		storer:    storer,
	}
}

func (s Service) Run() <-chan error {
	errChan := make(chan error)
	s.node.RegisterProtocol("/kdmgetvalue", func(c transport.Conn) {
		// Decode request.
		var req Request
		decoder := gob.NewDecoder(c)
		err := decoder.Decode(&req)
		if err != nil {
			errChan <- err
			sendResponse(c, Response{Err: ErrInvalidRequest})
			return
		}

		// Validate reaquest.
		if !s.isValidRequest(req) {
			sendResponse(c, Response{Err: ErrInvalidRequest})
			return
		}

		// Check if we have value localy.
		value, err := s.storer.Get(req.Key)
		if err == nil {
			err := sendResponse(c, Response{Value: value})
			if err != nil {
				errChan <- err
				return
			}
			return
		}
		if !errors.Is(err, storage.ErrNotFound) {
			errChan <- err
			sendResponse(c, Response{Err: ErrInternalServerError})
			return
		}

		// Get the closest nodes we know.
		peers, dis, err := s.peerstore.GetClosestPeers(req.Key, req.K)
		if err != nil {
			errChan <- err
			sendResponse(c, Response{Err: ErrInternalServerError})
			return
		}
		if len(peers) != len(dis) {
			errChan <- errors.New("number of distences and peers returned from peerstore are diffrent")
			sendResponse(c, Response{Err: ErrInternalServerError})
			return
		}

		// Convert data and filter out nodes that are not closer then we are.
		thisNodeDistance, err := s.peerstore.Distance(s.node.ID(), req.Key)
		if err != nil {
			errChan <- err
			sendResponse(c, Response{Err: ErrInternalServerError})
			return
		}

		nodes := make([]Node, 0, len(peers))
		for i := 0; i < len(peers); i++ {
			if dis[i].Cmp(thisNodeDistance) > 0 {
				continue
			}

			nodes = append(nodes, Node{peers[i].ID(), dis[i], peers[i].PublicAddr()})
		}

		err = sendResponse(c, Response{ClosestNodes: nodes})
		if err != nil {
			errChan <- err
		}
	})

	return errChan
}

func (s Service) Do(ctx context.Context, req Request, peer peer.Peer) (Response, error) {
	conn, err := s.node.DialPeerUsingProcol(ctx, "/kdmgetvalue", peer)
	if err != nil {
		return Response{}, fmt.Errorf("%w: %w", ErrUnableToReachPeer, err)
	}
	defer conn.Close()

	fmt.Println("sending req sdfl...")
	encoder := gob.NewEncoder(conn)
	err = encoder.Encode(req)
	if err != nil {
		return Response{}, fmt.Errorf("%w: %w", ErrUnableToReachPeer, err)
	}

	fmt.Println("wating on response...")
	var response Response
	decoder := gob.NewDecoder(conn)
	err = decoder.Decode(&response)
	if err != nil {
		return Response{}, fmt.Errorf("%w: %w", ErrInvalidResponse, err)
	}

	return response, response.Err
}

func (s Service) isValidRequest(req Request) bool {
	return len(req.Key) == len(s.node.ID())
}

func sendResponse(c transport.Conn, r Response) error {
	var msg bytes.Buffer
	encoder := gob.NewEncoder(&msg)
	err := encoder.Encode(r)
	if err != nil {
		return err
	}

	_, err = c.Write(msg.Bytes())
	return err
}
