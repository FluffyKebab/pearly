package transform

import (
	"crypto/rand"
	"testing"
	"time"

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

func testTransformerFlip(data []byte) ([]byte, error) {
	res := make([]byte, len(data))
	for i := 0; i < len(data); i++ {
		res[i] = ^data[i]
	}

	return res, nil
}

func TestTransformerSingleSmallMessage(t *testing.T) {
	conn1To2Underlaying := &mockedConn{
		buffer: make([]byte, 20*1048),
	}
	conn2To1Underlaying := &mockedConn{}

	conn1To2 := NewConn(conn1To2Underlaying, testTransformerFlip, testTransformerFlip, time.Second)
	conn2To1 := NewConn(conn2To1Underlaying, testTransformerFlip, testTransformerFlip, time.Second)

	msgSent := make([]byte, 10)
	_, err := rand.Read(msgSent)
	require.NoError(t, err)

	n, err := conn1To2.Write(msgSent)
	require.NoError(t, err)
	require.Equal(t, len(msgSent), n)

	// Simulate the data being sent form node 1 to 2.
	conn2To1Underlaying.buffer = conn1To2Underlaying.buffer
	conn2To1Underlaying.pos = conn1To2Underlaying.pos

	msgRecived := make([]byte, len(msgSent))
	var totalRead int
	for totalRead < len(msgSent) {
		curRead, err := conn2To1.Read(msgRecived[totalRead:])
		require.NoError(t, err)
		totalRead += curRead
	}

	require.Equal(t, msgSent, msgRecived)
}
