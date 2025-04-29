package onion

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/FluffyKebab/pearly/onion"
	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/transport"
)

var (
	ErrNotEnoughRelayServers = errors.New("to few relay servers found")
	ErrConnectionFailed      = errors.New("connection failed")
)

type Transport struct {
	Underlying   transport.Transport
	Client       onion.Client
	RelayServers peer.Store
	NumRelays    int
	NumRetries   int
}

var _ transport.Transport = &Transport{}

func (t *Transport) Dial(ctx context.Context, p peer.Peer) (transport.Conn, error) {
	peers := t.RelayServers.Peers()
	if len(peers) < t.NumRelays {
		return nil, fmt.Errorf("%w: have: %v, need: %v", ErrNotEnoughRelayServers, len(peers), t.NumRelays)
	}

	seed := uint64(time.Now().Nanosecond())
	r := rand.New(rand.NewPCG(seed, seed))
	relays := make([]peer.Peer, t.NumRelays+1)
	for i := 0; i < len(relays); i++ {
		selected := r.IntN(len(peers))
		relays[i] = peers[selected]
		relays = remove(peers, selected)
	}
	relays[len(relays)-1] = p

	errs := make([]error, 0)
	for i := 0; i < t.NumRetries; i++ {
		c, failed, err := t.Client.EstablishCircuit(ctx, relays)
		if err == nil {
			return c, nil
		}

		errs = append(errs, err)

		relays = remove(relays, failed)
		if len(peers) == 0 {
			return nil, constructErr(errs)
		}
		if failed >= len(relays) {
			return nil, fmt.Errorf("invalid failed index returned from onion client")
		}

		selected := r.IntN(len(peers))
		relays[failed] = peers[selected]
		peers = remove(peers, selected)
	}

	return nil, constructErr(errs)
}

func (t *Transport) Listen(ctx context.Context) (<-chan transport.Conn, <-chan error, error) {
	return t.Underlying.Listen(ctx)
}

func (t *Transport) ListenAddr() string {
	return t.Underlying.ListenAddr()
}

func constructErr(errs []error) error {
	finalErr := errors.New("")
	for _, err := range errs {
		finalErr = fmt.Errorf("%w, %w", err, finalErr)
	}

	return fmt.Errorf("%w: [%w]", ErrConnectionFailed, finalErr)
}

func remove(s []peer.Peer, i int) []peer.Peer {
	return append(s[:i], s[i+1:]...)

}
