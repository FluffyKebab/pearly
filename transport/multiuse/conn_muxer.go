package multiuse

import (
	"encoding/gob"
	"errors"

	"github.com/FluffyKebab/pearly/transport"
	"github.com/google/uuid"
)

var ErrInvalidConnID = errors.New("invalid connection id")

type connMuxerPayload struct {
	ConnID  string
	Data    []byte
	Closing bool
}

type ConnMuxer struct {
	underlayingConn transport.Conn
	peerID          []byte
	encoder         *gob.Encoder
	decoder         *gob.Decoder
	conns           map[string]*Conn
	done            chan struct{}
}

func NewConnMuxer(
	underlaying transport.Conn,
	peerID []byte,
	t *Transport,
) *ConnMuxer {
	cm := &ConnMuxer{
		underlayingConn: underlaying,
		encoder:         gob.NewEncoder(underlaying),
		decoder:         gob.NewDecoder(underlaying),
		conns:           make(map[string]*Conn, 0),
		done:            make(chan struct{}),
	}

	go func() {
		for {
			select {
			case <-cm.done:
				return
			default:
			}

			newConn, err := cm.handleNewPayload()
			if err != nil {
				t.errChan <- err
			}
			if newConn != nil {
				t.connChan <- newConn
			}
		}
	}()

	return cm
}

func (cm *ConnMuxer) handleNewPayload() (*Conn, error) {
	var payload connMuxerPayload
	err := cm.decoder.Decode(&payload)
	if err != nil {
		return nil, err
	}

	// The connection allready exist.
	if conn, ok := cm.conns[payload.ConnID]; ok {
		if payload.Closing {
			cm.conns[payload.ConnID] = nil
		}

		conn.writeUread(payload.Data, payload.Closing)
		return nil, nil
	}

	// Peer wants to open a new connection.
	newConn := NewConn(cm, payload.ConnID)
	cm.conns[payload.ConnID] = newConn
	newConn.writeUread(payload.Data, false)
	return newConn, nil
}

func (cm *ConnMuxer) NewConn() (*Conn, error) {
	id := uuid.New().String()
	newConn := NewConn(cm, id)
	cm.conns[id] = newConn

	return newConn, nil
}

func (cm *ConnMuxer) WriteToConnWithID(id string, data []byte) error {
	if _, ok := cm.conns[id]; !ok {
		return ErrInvalidConnID
	}

	return cm.encoder.Encode(connMuxerPayload{
		ConnID: id,
		Data:   data,
	})
}

func (cm *ConnMuxer) CloseConnWithID(id string) error {
	if _, ok := cm.conns[id]; !ok {
		return ErrInvalidConnID
	}

	return cm.encoder.Encode(connMuxerPayload{
		ConnID:  id,
		Closing: true,
	})
}

// Close closes the underlaying connection of the muxer.
func (cm *ConnMuxer) Close() error {
	<-cm.done
	return cm.underlayingConn.Close()
}
