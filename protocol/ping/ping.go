package ping

import (
	"bufio"
	"context"
	"time"

	"github.com/FluffyKebab/pearly/node"
	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/transport"
)

type Service struct {
	node node.Node
}

func (s Service) Do(ctx context.Context, peer peer.Peer) (time.Duration, error) {
	c, err := s.node.DialPeerUsingProcol(ctx, "/ping", peer)
	if err != nil {
		return 0, err
	}
	defer c.Close()

	t1 := time.Now()
	_, err = c.Write([]byte("ping\n"))
	if err != nil {
		return 0, err
	}

	_, _, err = bufio.NewReader(c).ReadLine()
	if err != nil {
		return 0, err
	}

	return time.Since(t1), nil
}

func (s Service) Run() {
	s.node.RegisterProtocol("/ping", func(c transport.Conn) error {
		_, _, err := bufio.NewReader(c).ReadLine()
		if err != nil {
			return err
		}

		_, err = c.Write([]byte("pong\n"))
		if err != nil {
			return err
		}

		return nil
	})
}

func Register(n node.Node) Service {
	return Service{n}
}
