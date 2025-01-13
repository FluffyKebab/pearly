package kdmgetvalue

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

type Request struct {
	Key []byte
	K   int
}

type Response struct {
	Value        []byte
	ClosestNodes []Node
}

type Node struct {
	ID         []byte
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
		fmt.Println("handling kdm")
		var req Request
		decoder := gob.NewDecoder(c)
		err := decoder.Decode(&req)
		if err != nil {
			errChan <- err
			return
		}

		fmt.Println("decoded ")

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
			return
		}

		peers, err := s.peerstore.GetClosestPeers(req.Key, req.K)
		if err != nil {
			errChan <- err
			return
		}

		nodes := make([]Node, len(peers))
		for i := 0; i < len(peers); i++ {
			nodes[i] = Node{peers[i].ID(), peers[i].PublicAddr()}
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
		return Response{}, err
	}
	defer conn.Close()

	fmt.Println("sending req sdfl...")
	encoder := gob.NewEncoder(conn)
	err = encoder.Encode(req)
	if err != nil {
		return Response{}, err
	}

	fmt.Println("wating on response...")
	var response Response
	decoder := gob.NewDecoder(conn)
	err = decoder.Decode(&response)
	return response, err
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
