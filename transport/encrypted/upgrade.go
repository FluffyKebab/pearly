package encrypted

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"encoding/gob"
	"fmt"
	"strings"

	"github.com/FluffyKebab/pearly/transport"
)

type upgraderPayload struct {
	ID            []byte
	PublicKey     []byte
	ListeningPort string
}

func (t Transport) upgradeConn(c transport.Conn) (*Conn, error) {
	encoder := gob.NewEncoder(c)
	decoder := gob.NewDecoder(c)

	err := encoder.Encode(upgraderPayload{
		ID:            t.id,
		PublicKey:     t.publicKey,
		ListeningPort: strings.Split(t.ListenAddr(), ":")[1],
	})
	if err != nil {
		return nil, err
	}

	var peerData upgraderPayload
	err = decoder.Decode(&peerData)
	if err != nil {
		return nil, err
	}

	peerPubKey, err := x509.ParsePKCS1PublicKey(peerData.PublicKey)
	if err != nil {
		return nil, err
	}

	acctuallID := sha256.Sum256(peerData.PublicKey)
	if !bytes.Equal(peerData.ID, acctuallID[:]) {
		return nil, fmt.Errorf("peer public key does not match with their node ID")
	}

	return NewConn(c, peerPubKey, t.privateKey, decoder, encoder, peerData.ID, peerData.ListeningPort), nil
}
