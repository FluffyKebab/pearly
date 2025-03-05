package onion

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"time"

	"github.com/FluffyKebab/pearly/node"
	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/transport"
)

var ErrInvalidRequest = errors.New("invalid request")

const (
	OnionProtoID     = "/onion"
	_sucssesResponse = "sucsess"
)

type Service struct {
	Node    node.Node
	Pealer  Pealer
	Wrapper Wrapper
}

func (s *Service) Run() {
	s.Node.RegisterProtocol(OnionProtoID, func(c transport.Conn) error {
		previousConnReader, err := s.Pealer.PealConn(c)
		if err != nil {
			c.Close()
			return err
		}

		previousConnWriter, err := s.Wrapper.WrapConn(c)
		if err != nil {
			c.Close()
			return err
		}

		req, err := readRequest(previousConnReader)
		if err != nil {
			c.Close()
			return err
		}

		nextConn, err := s.Node.DialPeer(
			context.Background(),
			peer.New(nil, req.NextNodeAddr),
		)
		if err != nil {
			sendFail(previousConnWriter, err)
			c.Close()
			return err
		}

		err = sendSuccses(previousConnWriter)
		if err != nil {
			c.Close()
			return err
		}

		s.handleRelay(previousConnReader, previousConnWriter, c, nextConn, req.MaxRandomWaitTime)
		return nil
	})
}

func (s *Service) handleRelay(
	prevConnReader io.Reader,
	prevConnWriter io.Writer,
	prevConnCloser io.Closer,
	nextConn transport.Conn,
	maxRandomWaitTime time.Duration,
) {
	go func() {
		err := copyWithRandomWait(prevConnWriter, nextConn, maxRandomWaitTime)
		if err != nil {
			s.Node.SendError(fmt.Errorf("onion relay: %w", err))
		}
		prevConnCloser.Close()
	}()
	go func() {
		err := copyWithRandomWait(nextConn, prevConnReader, maxRandomWaitTime)
		if err != nil {
			s.Node.SendError(fmt.Errorf("onion relay: %w", err))
		}
		prevConnCloser.Close()
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

func copyWithRandomWait(dst io.Writer, src io.Reader, wait time.Duration) error {
	size := 32 * 1024
	buf := make([]byte, size)
	var err error

	for {
		time.Sleep(time.Duration((rand.Int64N(int64(wait)))))

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
