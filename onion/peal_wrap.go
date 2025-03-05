package onion

import (
	"crypto/rsa"
	"encoding/gob"
	"io"

	"github.com/FluffyKebab/pearly/transport"
	"github.com/FluffyKebab/pearly/transport/encrypted"
)

type Pealer interface {
	PealConn(conn transport.Conn) (io.Reader, error)
}

type Wrapper interface {
	WrapConn(conn transport.Conn) (io.Writer, error)
}

type RSAPealer struct {
	PrivateKey *rsa.PrivateKey
}

func NewPealer(privKey *rsa.PrivateKey) *RSAPealer {
	return &RSAPealer{
		PrivateKey: privKey,
	}
}

func (p *RSAPealer) PealConn(conn transport.Conn) (io.Reader, error) {
	return encrypted.NewConn(
		conn,
		&p.PrivateKey.PublicKey,
		p.PrivateKey,
		gob.NewDecoder(conn),
		gob.NewEncoder(conn),
		nil,
		"",
	), nil
}

type RSAWrapper struct {
	PublicKey *rsa.PublicKey
}

func NewWrapper(pubKey *rsa.PublicKey) *RSAWrapper {
	return &RSAWrapper{
		PublicKey: pubKey,
	}
}

func (p *RSAWrapper) WrapConn(conn transport.Conn) (io.Writer, error) {
	return encrypted.NewConn(
		conn,
		p.PublicKey,
		nil,
		gob.NewDecoder(conn),
		gob.NewEncoder(conn),
		nil,
		"",
	), nil
}
