package crypto

import (
	"bytes"
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/FluffyKebab/pearly/peer"
	"github.com/FluffyKebab/pearly/testutil"
	"github.com/FluffyKebab/pearly/transport"
	"github.com/FluffyKebab/pearly/transport/tcp"
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

	var msg [1028]byte
	rand.Read(msg[:])
	_, err = s.Write(msg[:])
	require.NoError(t, err)

	var msgGotten [1028]byte
	n, err := s.Read(msgGotten[:len(msg)])

	require.NoError(t, err)
	require.Equal(t, msg[:], msgGotten[:n])
}

func TestStreamReadBeforeWrite(t *testing.T) {
	port, err := testutil.GetAvilablePort()
	require.NoError(t, err)
	c := tcp.New(port)
	listner, _, _ := c.Listen(context.Background())
	conn, err := c.Dial(context.Background(), peer.New(nil, "localhost:"+port))
	require.NoError(t, err)

	secretKey1 := NewSymetricEncryptionSecretKey()
	secretKey2 := NewSymetricEncryptionSecretKey()
	s, err := NewEncryptionStream(secretKey1, conn)
	require.NoError(t, err)
	s, err = NewEncryptionStream(secretKey2, s)
	require.NoError(t, err)

	var msg [1028]byte
	rand.Read(msg[:])
	done := make(chan struct{})
	go func() {
		conn := <-listner
		s, err := NewEncryptionStream(secretKey1, conn)
		require.NoError(t, err)
		s, err = NewEncryptionStream(secretKey2, s)
		require.NoError(t, err)

		var msgGotten [1028]byte
		n, err := s.Read(msgGotten[:len(msg)])

		require.NoError(t, err)
		require.Equal(t, msg[:], msgGotten[:n])
		done <- struct{}{}
	}()

	time.Sleep(time.Second)

	_, err = s.Write(msg[:])
	require.NoError(t, err)
	<-done
}
