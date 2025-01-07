package ping

import (
	"bufio"
	"context"
	"time"

	"github.com/FluffyKebab/pearly/node"
	"github.com/FluffyKebab/pearly/transport"
)

func Register(n node.Node) (func(ctx context.Context, nodeID string) (time.Duration, error), <-chan error) {
	errChan := make(chan error)
	n.RegisterProtocol("/ping", func(c transport.Conn) {
		_, _, err := bufio.NewReader(c).ReadLine()
		if err != nil {
			errChan <- err
			return
		}

		_, err = c.Write([]byte("pong\n"))
		if err != nil {
			errChan <- err
			return
		}
	})

	return func(ctx context.Context, nodeID string) (time.Duration, error) {
		c, err := n.DialPeerUsingProcol(ctx, "/ping", nodeID)
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
	}, errChan
}
