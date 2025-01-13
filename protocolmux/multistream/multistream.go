package multistream

import (
	"context"
	"fmt"
	"io"

	"github.com/FluffyKebab/pearly/protocolmux"
	"github.com/FluffyKebab/pearly/transport"
	ms "github.com/multiformats/go-multistream"
)

type Muxer struct {
	mux *ms.MultistreamMuxer[string]
}

var _ protocolmux.Muxer = Muxer{}

func NewMuxer() Muxer {
	return Muxer{
		mux: ms.NewMultistreamMuxer[string](),
	}
}

func (m Muxer) RegisterProtocol(protoID string, handler func(transport.Conn)) {
	m.mux.AddHandler(protoID, func(_ string, rwc io.ReadWriteCloser) error {
		handler(rwc)
		return nil
	})
}

func (m Muxer) SelectProtocol(ctx context.Context, protoID string, c transport.Conn) error {
	done := make(chan error)
	go func() {
		err := ms.SelectProtoOrFail(protoID, c)
		done <- err
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("select protocol: %w", context.Canceled)
	}
}

func (m Muxer) HandleConn(c transport.Conn) error {
	return m.mux.Handle(c)
}
