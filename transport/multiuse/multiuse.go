package multiuse

import (
	"context"
	"errors"
	"time"

	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/transport"
)

type Transport struct {
	underlaying   transport.Transport
	keepALiveTime time.Duration
	openConns     map[string]*ConnMuxer
	errChan       chan error
	connChan      chan transport.Conn
}

var _ transport.Transport = &Transport{}

func New(underlaying transport.Transport, timeInbetweenKeepAlive time.Duration) *Transport {
	return &Transport{
		underlaying:   underlaying,
		keepALiveTime: timeInbetweenKeepAlive,
		openConns:     make(map[string]*ConnMuxer),
		errChan:       make(chan error),
		connChan:      make(chan transport.Conn),
	}
}

func (t *Transport) Dial(ctx context.Context, p peer.Peer) (transport.Conn, error) {
	if connMuxer, ok := t.openConns[string(p.ID())]; ok {
		return connMuxer.NewConn()
	}

	newConn, err := t.underlaying.Dial(ctx, p)
	if err != nil {
		return nil, err
	}

	newConnMuxer := NewConnMuxer(newConn, p.ID(), t)

	t.openConns[string(p.ID())] = newConnMuxer
	return newConnMuxer.NewConn()
}

func (t *Transport) Listen(ctx context.Context) (<-chan transport.Conn, <-chan error, error) {
	underlayingConnChan, errChan, err := t.underlaying.Listen(ctx)
	if err != nil {
		return nil, nil, err
	}

	go func() {
		for {
			select {
			case err := <-errChan:
				t.errChan <- err
			case conn := <-underlayingConnChan:
				if ider, ok := conn.(transport.RemoteIDHaver); ok {
					t.openConns[string(ider.RemoteID())] = NewConnMuxer(conn, ider.RemoteID(), t)
					continue
				}
				t.errChan <- errors.New("underlaying transport of multiuse must create remote id having connections. Try using encrypted instead")
			case <-ctx.Done():
				return
			}
		}
	}()

	return t.connChan, t.errChan, nil
}

func (t *Transport) ListenAddr() string {
	return t.underlaying.ListenAddr()
}
