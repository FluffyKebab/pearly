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

var _ transport.Transport = Transport{}

func New(port string) Transport {
	return Transport{
		port: port,
	}
}

func (t Transport) Listen(ctx context.Context) (<-chan transport.Conn, <-chan error, error) {
	connChan := make(chan transport.Conn)
	errChan := make(chan error)

	listener, err := net.Listen("tcp", "localhost:"+t.port)
	if err != nil {
		return nil, nil, err
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

			connChan <- connection{conn}
		}
	}()

	return connChan, errChan, nil
}

func (t Transport) Dial(ctx context.Context, p peer.Peer) (transport.Conn, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", p.PublicAddr())
	return connection{conn}, err
}

func (t Transport) ListenAddr() string {
	return "127.0.0.1:" + t.port
}
