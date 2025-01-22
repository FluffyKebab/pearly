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
	Err          string
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

func (s Service) Run() {
	s.node.RegisterProtocol("/kdmgetvalue", func(c transport.Conn) error {
		// Decode request.
		var req Request
		decoder := gob.NewDecoder(c)
		err := decoder.Decode(&req)
		if err != nil {
			sendResponse(c, Response{Err: ErrInvalidRequest.Error()})
			return err
		}

		// Validate request.
		if !s.isValidRequest(req) {
			sendResponse(c, Response{Err: ErrInvalidRequest.Error()})
			return nil
		}

		// Try to add connected peer to peerstore.
		if err := s.tryAddPeerToStore(c); err != nil {
			sendResponse(c, Response{Err: ErrInvalidRequest.Error()})
			return err
		}

		// Check if we have value localy.
		value, err := s.storer.Get(req.Key)
		if err == nil {
			return sendResponse(c, Response{Value: value})
		}
		if !errors.Is(err, storage.ErrNotFound) {
			sendResponse(c, Response{Err: ErrInternalServerError.Error()})
			return err
		}

		// Get the closest nodes we know.
		peers, dis, err := s.peerstore.GetClosestPeers(req.Key, req.K)
		if err != nil {
			sendResponse(c, Response{Err: ErrInternalServerError.Error()})
			return err
		}
		if len(peers) != len(dis) {
			sendResponse(c, Response{Err: ErrInternalServerError.Error()})
			return errors.New("number of distences and peers returned from peerstore are diffrent")
		}

		// Convert data and filter out nodes that are not closer then we are.
		thisNodeDistance, err := s.peerstore.Distance(s.node.ID(), req.Key)
		if err != nil {
			sendResponse(c, Response{Err: ErrInternalServerError.Error()})
			return err
		}

		nodes := make([]Node, 0, len(peers))
		for i := 0; i < len(peers); i++ {
			if dis[i].Cmp(thisNodeDistance) > 0 {
				continue
			}

			nodes = append(nodes, Node{peers[i].ID(), dis[i], peers[i].PublicAddr()})
		}

		return sendResponse(c, Response{ClosestNodes: nodes})
	})
}

func (s Service) Do(ctx context.Context, req Request, peer peer.Peer) (Response, error) {
	conn, err := s.node.DialPeerUsingProcol(ctx, "/kdmgetvalue", peer)
	if err != nil {
		return Response{}, fmt.Errorf("%w: %w", ErrUnableToReachPeer, err)
	}
	defer conn.Close()
	encoder := gob.NewEncoder(conn)
	err = encoder.Encode(req)
	if err != nil {
		return Response{}, fmt.Errorf("%w: %w", ErrUnableToReachPeer, err)
	}

	var response Response
	decoder := gob.NewDecoder(conn)
	err = decoder.Decode(&response)
	if err != nil {
		return Response{}, fmt.Errorf("%w: %w", ErrInvalidResponse, err)
	}

	return response, convertToError(response.Err)
}

func (s Service) isValidRequest(req Request) bool {
	return len(req.Key) == len(s.node.ID())
}

func (s Service) tryAddPeerToStore(c transport.Conn) error {
	ider, ok := c.(transport.RemoteIDHaver)
	if !ok {
		return nil
	}

	adder, ok := c.(transport.RemoteAddrHaver)
	if !ok {
		return nil
	}

	err := s.peerstore.AddPeer(peer.New(ider.RemoteID(), adder.RemoteAddr()))
	if err != nil && !errors.Is(err, peer.ErrNoSpaceToStorePeer) {
		return err
	}

	return nil
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

func convertToError(s string) error {
	if s == "" {
		return nil
	}

	if ErrInvalidResponse.Error() == s {
		return ErrInvalidResponse
	}
	if ErrInvalidRequest.Error() == s {
		return ErrInvalidRequest
	}
	if ErrInternalServerError.Error() == s {
		return ErrInternalServerError
	}

	return errors.New(s)
}
