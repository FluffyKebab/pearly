package encrypted

import (
	"bytes"
	"crypto/x509"
	"encoding/gob"

	"github.com/FluffyKebab/pearly/transport"
)

func (t Transport) upgradeConn(c transport.Conn) (Conn, error) {
	err := sendPayload(c, t.id, t.publicKey)
	if err != nil {
		return Conn{}, err
	}

	peerData, err := recivePayload(c)
	if err != nil {
		return Conn{}, err
	}

	peerPubKey, err := x509.ParsePKCS1PublicKey(peerData.PublicKey)
	if err != nil {
		return Conn{}, err
	}

	return NewConn(c, peerPubKey, t.privateKey), nil
}

type upgraderPayload struct {
	ID        []byte
	PublicKey []byte
}

func sendPayload(conn transport.Conn, id []byte, pubKey []byte) error {
	var msg bytes.Buffer
	encoder := gob.NewEncoder(&msg)
	err := encoder.Encode(upgraderPayload{
		ID:        id,
		PublicKey: pubKey,
	})
	if err != nil {
		return err
	}

	_, err = conn.Write(msg.Bytes())
	return err
}

func recivePayload(conn transport.Conn) (upgraderPayload, error) {
	var msg upgraderPayload
	decoder := gob.NewDecoder(conn)
	err := decoder.Decode(&msg)
	return msg, err
}
