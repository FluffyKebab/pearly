package dhtpeer

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/FluffyKebab/pearly/peer"
	"github.com/stretchr/testify/require"
)

func TestPeerStore(t *testing.T) {
	s := NewStore([]byte{0b11111111}, 2)

	err := s.AddPeer(peer.New([]byte{0b11000000}, ""))
	require.NoError(t, err)

	err = s.AddPeer(peer.New([]byte{0b11000001}, ""))
	require.NoError(t, err)

	err = s.AddPeer(peer.New([]byte{0b11000000}, ""))
	require.NoError(t, err)

	err = s.AddPeer(peer.New([]byte{0b00000001}, ""))
	require.NoError(t, err)

	require.True(t, bytes.Equal([]byte{0b11000000}, s.buckets[2][0].ID()))
	require.True(t, bytes.Equal([]byte{0b11000001}, s.buckets[2][1].ID()))
	require.True(t, bytes.Equal([]byte{0b00000001}, s.buckets[0][0].ID()))

	closest, err := s.GetClosestPeers([]byte{0b11000100}, 1)
	require.NoError(t, err)
	require.True(t, bytes.Equal([]byte{0b11000000}, closest[0].ID()))

	err = s.RemovePeer(peer.New([]byte{0b11000000}, ""))
	require.NoError(t, err)
	require.True(t, bytes.Equal([]byte{0b11000001}, s.buckets[2][0].ID()))
	require.Nil(t, s.buckets[2][1])

}

func TestNumEqualBitsPrefix(t *testing.T) {
	require.Equal(t, 4, numEqualBitsPrefix(
		[]byte{0b01011110, 0b11111111},
		[]byte{0b01010000, 0b11011111}),
	)
}

func TestDistenceBetween(t *testing.T) {
	require.Equal(t, big.NewInt(65535), distenceBetween(
		[]byte{0b10101010, 0b11111111},
		[]byte{0b01010101, 0b00000000}),
	)
	require.Zero(t, big.NewInt(0).Cmp(distenceBetween(
		[]byte{0b10101010, 0b10101010},
		[]byte{0b10101010, 0b10101010}),
	))
}
