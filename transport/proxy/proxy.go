package proxy

import (
	"context"
	"fmt"

	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/protocol/proxy"
	"github.com/FluffyKebab/pearly/protocolmux"
	"github.com/FluffyKebab/pearly/transport"
)

type Transport struct {
	underlying    transport.Transport
	proxyServer   peer.Peer
	protocolMuxer protocolmux.Muxer
}

func New(
	underlying transport.Transport,
	protocolMuxer protocolmux.Muxer,
	proxyServer peer.Peer,
) Transport {
	return Transport{
		underlying:    underlying,
		proxyServer:   proxyServer,
		protocolMuxer: protocolMuxer,
	}
}

func (t Transport) Dial(ctx context.Context, peer peer.Peer) (transport.Conn, error) {
	c, err := t.underlying.Dial(ctx, t.proxyServer)
	if err != nil {
		return nil, fmt.Errorf("connecting to proxy server: %w", err)
	}

	err = t.protocolMuxer.SelectProtocol(ctx, "/proxy", c)
	if err != nil {
		return nil, err
	}

	err = proxy.SendRequest(c, peer.PublicAddr())
	return c, err
}

func (t Transport) Listen(ctx context.Context) (<-chan transport.Conn, <-chan error, error) {
	return t.underlying.Listen(ctx)
}

func (t Transport) ListenAddr() string {
	return t.underlying.ListenAddr()
}
