package crypto

import (
	"bytes"
	"testing"

	"github.com/FluffyKebab/pearly/transport"
	"github.com/stretchr/testify/require"
)

func TestSymetricalEncryption(t *testing.T) {
	encryptor, err := NewSymetricEncryption(NewSymetricEncryptionSecretKey())
	require.NoError(t, err)

	msg := "halo this is the mesage that should be the same anyways"
	chiper, err := encryptor.Encrypt([]byte(msg))
	require.NoError(t, err)

	plaintext, err := encryptor.Decrypt(chiper)
	require.NoError(t, err)
	require.Equal(t, msg, string(plaintext))
}

func TestStream(t *testing.T) {
	d := bytes.NewBuffer(make([]byte, 0))

	s, err := NewEncryptionStream(NewSymetricEncryptionSecretKey(), transport.NewConn(d, d, nil))
	require.NoError(t, err)

	msg := "wow msg from this to data yo"
	_, err = s.Write([]byte(msg))
	require.NoError(t, err)

	var msgGotten [512]byte
	n, err := s.Read(msgGotten[:len([]byte(msg))])

	require.NoError(t, err)
	require.Equal(t, msg, string(msgGotten[:n]))
}
