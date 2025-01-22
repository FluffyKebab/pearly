package encrypted

import (
	"crypto/x509"
	"encoding/gob"

	"github.com/FluffyKebab/pearly/transport"
)

type upgraderPayload struct {
	ID        []byte
	PublicKey []byte
}

func (t Transport) upgradeConn(c transport.Conn) (*Conn, error) {
	encoder := gob.NewEncoder(c)
	decoder := gob.NewDecoder(c)

	err := encoder.Encode(upgraderPayload{
		ID:        t.id,
		PublicKey: t.publicKey,
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

	// TODO: validate peer

	return NewConn(c, peerPubKey, t.privateKey, decoder, encoder, peerData.ID), nil
}
