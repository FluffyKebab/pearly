package tcp

import (
	"context"
	"errors"
	"net"

	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/transport"
)

type Transport struct {
	port string
}

func New(port string) Transport {
	return Transport{
		port: port,
	}
}

func (t Transport) Listen(ctx context.Context) (<-chan transport.Conn, <-chan error) {
	connChan := make(chan transport.Conn)
	errChan := make(chan error)

	listener, err := net.Listen("tcp", "localhost:"+t.port)
	if err != nil {
		errChan <- err
		return nil, errChan
	}

	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}

				errChan <- err
				continue
			}

			connChan <- conn
		}
	}()

	return connChan, errChan
}

func (t Transport) Dial(ctx context.Context, p peer.Peer) (transport.Conn, error) {
	var d net.Dialer
	return d.DialContext(ctx, "tcp", p.PublicAddr)
}
