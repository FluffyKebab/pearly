package encrypted

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type mockedConn struct {
	buffer   []byte
	pos      int
	isClosed bool
}

func (c *mockedConn) Read(p []byte) (n int, err error) {
	for i := 0; i < c.pos; i++ {
		p[i] = c.buffer[i]
	}

	return c.pos, nil
}

func (c *mockedConn) Write(p []byte) (n int, err error) {
	for i := 0; i < len(p); i++ {
		c.buffer[c.pos+i] = p[i]
	}
	c.pos += len(p)

	return len(p), nil
}

func (c *mockedConn) Close() error {
	c.isClosed = true
	return nil
}

func TestEncryptedConn(t *testing.T) {
	node1privKey, node1pubKey, err := generateKeyPair()
	require.NoError(t, err)
	node2privKey, node2pubKey, err := generateKeyPair()
	require.NoError(t, err)

	conn1To2 := &mockedConn{
		buffer: make([]byte, 1048),
	}
	conn2To1 := &mockedConn{
		buffer: make([]byte, 1048),
	}

	encryptedConn1to2 := NewConn(conn1To2, node2pubKey, node1privKey)
	encryptedConn2to1 := NewConn(conn2To1, node1pubKey, node2privKey)

	msg := []byte("hello from node 1, to node 2")
	_, err = encryptedConn1to2.Write(msg)
	require.NoError(t, err)

	// Simulate the data being sent form node 1 to 2.
	conn2To1.buffer = conn1To2.buffer
	conn2To1.pos = conn1To2.pos

	msgRecived := make([]byte, 1048)
	n, err := encryptedConn2to1.Read(msgRecived)
	require.NoError(t, err)

	require.Equal(t, msg, msgRecived[0:n])
}
