package onion

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"time"

	"github.com/FluffyKebab/pearly/crypto"
	"github.com/FluffyKebab/pearly/node"
	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/transport"
	"github.com/FluffyKebab/pearly/transport/transform"
)

var ErrInvalidRequest = errors.New("invalid request")

const (
	OnionProtoID     = "/onion"
	_sucssesResponse = "sucsess"
)

type Request struct {
	//
	SecretKey []byte

	// NextNodeAddr is the contact information for the next node in the circut.
	NextNodeAddr string

	// MaxRandomWaitTime is the maximum time this noe will wait to relay from
	// the next node in the circut to the previous one and vice versa.
	MaxRandomWaitTime time.Duration
}

type Service struct {
	Node               node.Node
	PublicKeyDecrypter crypto.Decrypter
}

func RegisterService(n node.Node) *Service {
	return &Service{
		Node: n,
	}
}

func (s *Service) Run() {
	s.Node.RegisterProtocol(OnionProtoID, func(c transport.Conn) error {
		if err := s.handler(c); err != nil {
			sendFail(c, err)
			c.Close()
			if !errors.Is(err, ErrInvalidRequest) {
				return err
			}
		}

		return nil
	})
}

func (s *Service) handler(c transport.Conn) error {
	previousConn := transform.NewConn(c, nil, nil)
	if s.PublicKeyDecrypter != nil {
		previousConn.Detransform = s.PublicKeyDecrypter.Decrypt
	}

	req, err := s.readRequest(previousConn)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidRequest, err)
	}

	nextConn, err := s.Node.DialPeer(
		context.Background(),
		peer.New(nil, req.NextNodeAddr),
	)
	if err != nil {
		return err
	}

	err = sendSuccses(previousConn)
	if err != nil {
		c.Close()
		return err
	}

	upgradeToSymetriclyEncrypted(previousConn, req.SecretKey)
	s.handleRelay(previousConn, nextConn, req.MaxRandomWaitTime)
	return nil
}

func (s *Service) readRequest(c transport.Conn) (Request, error) {
	requestData := make([]byte, 1024)
	_, err := c.Read(requestData)
	if err != nil {
		return Request{}, err
	}

	var req Request
	err = gob.NewDecoder(bytes.NewBuffer(requestData)).Decode(&req)
	return req, err
}

func (s *Service) handleRelay(
	prevConn transport.Conn,
	nextConn transport.Conn,
	maxRandomWaitTime time.Duration,
) {
	go func() {
		err := copyWithRandomWait(prevConn, nextConn, maxRandomWaitTime)
		if err != nil {
			s.Node.SendError(fmt.Errorf("onion relay: %w", err))
		}
		prevConn.Close()
	}()
	go func() {
		err := copyWithRandomWait(nextConn, prevConn, maxRandomWaitTime)
		if err != nil {
			s.Node.SendError(fmt.Errorf("onion relay: %w", err))
		}
		nextConn.Close()
	}()
}

// sendFail informes the circut creator that the exist node failed to establish
// a connection with the node specified in the request.
func sendFail(w io.Writer, err error) {
	w.Write([]byte(err.Error()))
}

func sendSuccses(w io.Writer) error {
	_, err := w.Write([]byte(_sucssesResponse))
	return err
}

func upgradeToSymetriclyEncrypted(c *transform.Conn, secretKey []byte) error {
	encryption, err := crypto.NewSymetricEncryption(secretKey)
	if err != nil {
		return err
	}

	c.Transform = func(b []byte) ([]byte, error) {
		res, err := encryption.Encrypt(b)
		return res, err
	}
	c.Detransform = func(b []byte) ([]byte, error) {
		res, err := encryption.Decrypt(b)
		return res, err
	}
	return nil
}

func copyWithRandomWait(dst io.Writer, src io.Reader, wait time.Duration) error {
	size := 12 * 1024
	buf := make([]byte, size)
	var err error

	for {
		if int64(wait) > 0 {
			time.Sleep(time.Duration((rand.Int64N(int64(wait)))))
		}

		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = fmt.Errorf("invalid write")
				}
			}

			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = fmt.Errorf("invalid write")
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return err
}
