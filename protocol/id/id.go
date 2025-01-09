package id

import (
	"bytes"
	"context"
	"encoding/gob"

	"github.com/FluffyKebab/pearly/node"
	"github.com/FluffyKebab/pearly/transport"
)

type Service struct {
	ErrChan  <-chan error
	PeerChan <-chan ConnectedPeer

	node      node.Node
	publicKey []byte
	id        []byte
}

func (s Service) Do(ctx context.Context, conn transport.Conn) (peerID []byte, peerPubKey []byte, err error) {
	err = s.node.ChangeProtocol(ctx, "/id", conn)
	if err != nil {
		return nil, nil, err
	}

	err = sendPayload(conn, s.id, s.publicKey)
	if err != nil {
		return nil, nil, err
	}

	peerData, err := recivePayload(conn)
	return peerData.ID, peerData.PublicKey, nil
}

func Register(n node.Node, publicKey []byte, nodeID []byte) Service {
	errChan := make(chan error)
	peerChan := make(chan ConnectedPeer)
	s := Service{
		ErrChan:   errChan,
		PeerChan:  peerChan,
		node:      n,
		publicKey: publicKey,
		id:        nodeID,
	}

	n.RegisterProtocol("/id", func(conn transport.Conn) {
		peerData, err := recivePayload(conn)
		if err != nil {
			conn.Close()
			errChan <- err
			return
		}

		err = sendPayload(conn, nodeID, publicKey)
		if err != nil {
			conn.Close()
			errChan <- err
			return
		}

		peerChan <- ConnectedPeer{
			Conn:      conn,
			ID:        peerData.ID,
			PublicKey: peerData.PublicKey,
		}
	})

	return s
}

type ConnectedPeer struct {
	Conn      transport.Conn
	ID        []byte
	PublicKey []byte
}

type payload struct {
	ID        []byte
	PublicKey []byte
}

func sendPayload(conn transport.Conn, id []byte, pubKey []byte) error {
	var msg bytes.Buffer
	encoder := gob.NewEncoder(&msg)
	err := encoder.Encode(payload{
		ID:        id,
		PublicKey: pubKey,
	})
	if err != nil {
		return err
	}

	_, err = conn.Write(msg.Bytes())
	return err
}

func recivePayload(conn transport.Conn) (payload, error) {
	var msg payload
	decoder := gob.NewDecoder(conn)
	err := decoder.Decode(&msg)
	return msg, err
}
