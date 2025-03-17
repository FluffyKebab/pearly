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
	Value         []byte
	NodeContacted Node
	ClosestNodes  []Node
	Err           string
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
		var req Request
		err := gob.NewDecoder(c).Decode(&req)
		if err != nil {
			sendResponse(c, Response{Err: ErrInvalidRequest.Error()})
			return nil
		}

		res, err := s.HandleRequest(req)
		if err != nil {
			sendResponse(c, Response{Err: err.Error()})
			if !errors.Is(err, ErrInvalidRequest) {
				return err
			}
			return nil
		}

		if err := s.tryAddPeerToStore(c); err != nil {
			sendResponse(c, Response{Err: err.Error()})
			return nil
		}

		return sendResponse(c, res)
	})
}

func (s Service) HandleRequest(req Request) (Response, error) {
	// Validate request.
	if !s.isValidRequest(req) {
		return Response{}, ErrInvalidRequest
	}

	// Check if we have value localy.
	value, err := s.storer.Get(req.Key)
	if err == nil {
		return Response{
			Value: value,
			NodeContacted: Node{
				ID:         s.node.ID(),
				PublicAddr: s.node.Transport().ListenAddr(),
			},
		}, nil
	}
	if !errors.Is(err, storage.ErrNotFound) {
		return Response{}, fmt.Errorf("%w: %w", ErrInternalServerError, err)
	}

	// Get the closest nodes we know.
	peers, dis, err := s.peerstore.GetClosestPeers(req.Key, req.K)
	if err != nil {
		return Response{}, fmt.Errorf("%w: %w", ErrInternalServerError, err)
	}
	if len(peers) != len(dis) {
		return Response{}, fmt.Errorf(
			"%w: number of distences and peers returned from peerstore are diffrent",
			ErrInternalServerError,
		)
	}

	// Convert data and calculate the distence from this node to the key.
	thisNodeDistance, err := s.peerstore.Distance(s.node.ID(), req.Key)
	if err != nil {
		return Response{}, fmt.Errorf("%w: %w", ErrInternalServerError, err)
	}

	nodes := make([]Node, 0, len(peers))
	for i := 0; i < len(peers); i++ {
		nodes = append(nodes, Node{peers[i].ID(), dis[i], peers[i].PublicAddr()})
	}

	return Response{
		NodeContacted: Node{
			ID:         s.node.ID(),
			Distance:   thisNodeDistance,
			PublicAddr: s.node.Transport().ListenAddr(),
		},
		ClosestNodes: nodes,
	}, nil
}

func (s Service) Do(ctx context.Context, req Request, p peer.Peer) (Response, error) {
	conn, err := s.node.DialPeerUsingProcol(ctx, "/kdmgetvalue", p)
	if err != nil {
		return Response{}, fmt.Errorf("%w: %w", ErrUnableToReachPeer, err)
	}
	defer conn.Close()

	err = gob.NewEncoder(conn).Encode(req)
	if err != nil {
		return Response{}, fmt.Errorf("%w: %w", ErrUnableToReachPeer, err)
	}

	var response Response
	err = gob.NewDecoder(conn).Decode(&response)
	if err != nil {
		return Response{}, fmt.Errorf("%w: %w", ErrInvalidResponse, err)
	}
	if !s.isValidResponse(response, conn) {
		return Response{}, ErrInvalidResponse
	}

	return response, convertToError(response.Err)
}

func (s Service) isValidRequest(req Request) bool {
	return len(req.Key) == len(s.node.ID())
}

func (s Service) isValidResponse(res Response, conn transport.Conn) bool {
	remoteId, ok := conn.(transport.RemoteIDHaver)
	if !ok {
		return true
	}

	return bytes.Equal(remoteId.RemoteID(), res.NodeContacted.ID)
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
	encoder := gob.NewEncoder(c)
	return encoder.Encode(r)
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
