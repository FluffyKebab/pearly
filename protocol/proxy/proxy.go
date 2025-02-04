package proxy

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/FluffyKebab/pearly/node"
	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/transport"
)

const _sucsess = "sucses"

type Service struct {
	node node.Node

	Timeout time.Duration
	ErrChan chan error
}

func Register(node node.Node) Service {
	return Service{
		node:    node,
		ErrChan: make(chan error),
		Timeout: time.Second * 10,
	}
}

func (s Service) Run() {
	s.node.RegisterProtocol("/proxy", func(senderConn transport.Conn) error {
		reciverAddr, err := readUntilNewline(senderConn)
		if err != nil {
			sendProxyResponse(senderConn, "contacting proxy server: invalid request")
			senderConn.Close()
			return err
		}

		ctx, cancelFunc := context.WithTimeout(context.Background(), s.Timeout)
		defer cancelFunc()
		reciverConn, err := s.node.DialPeer(ctx, peer.New(nil, reciverAddr))
		if err != nil {
			sendProxyResponse(senderConn, "contacting proxy server: failed to contact peer")
			senderConn.Close()
			return err
		}

		go func() {
			_, err := io.Copy(senderConn, reciverConn)
			if err != nil {
				s.ErrChan <- err
			}
			senderConn.Close()
		}()
		go func() {
			_, err := io.Copy(reciverConn, senderConn)
			if err != nil {
				s.ErrChan <- err
			}
			reciverConn.Close()
		}()

		return sendProxyResponse(senderConn, _sucsess)
	})
}

func SendRequest(c transport.Conn, reciverAddr string) error {
	_, err := c.Write([]byte(reciverAddr + "\n"))
	if err != nil {
		return err
	}

	return readProxyResponse(c)
}

func readProxyResponse(c transport.Conn) error {
	response, err := readUntilNewline(c)
	if err != nil {
		return err
	}
	if response != _sucsess {
		return errors.New(response)
	}

	return nil
}

func sendProxyResponse(c transport.Conn, s string) error {
	_, err := c.Write([]byte(s + "\n"))
	return err
}

func readUntilNewline(r io.Reader) (string, error) {
	buf := make([]byte, 1)
	res := make([]byte, 128)
	n := 0
	for {
		_, err := r.Read(buf)
		if err != nil {
			if err == io.EOF {
				return "", io.ErrUnexpectedEOF
			}
			return "", err
		}
		if buf[0] == '\n' {
			break
		}
		res[n] = buf[0]
		n++
	}

	return string(res[0:n]), nil
}
