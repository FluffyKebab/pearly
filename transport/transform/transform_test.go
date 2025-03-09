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
	read     int
	isClosed bool
}

func (c *mockedConn) Read(p []byte) (int, error) {
	var n int
	for c.read < c.pos && n < len(p) {
		p[n] = c.buffer[c.read]
		c.read++
		n++
	}

	return n, nil
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

func testTransformerNone(data []byte) ([]byte, error) {
	res := make([]byte, len(data))
	copy(res, data)
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

func TestTransformerMultipleSmallMessages(t *testing.T) {
	conn1To2Underlaying := &mockedConn{
		buffer: make([]byte, 20*1048),
	}
	conn2To1Underlaying := &mockedConn{}

	conn1To2 := NewConn(conn1To2Underlaying, testTransformerFlip, testTransformerFlip, time.Second)
	conn2To1 := NewConn(conn2To1Underlaying, testTransformerFlip, testTransformerFlip, time.Second)

	msgs := make([][]byte, 10)
	for i := 0; i < len(msgs); i++ {
		msgs[i] = make([]byte, 10)
		_, err := rand.Read(msgs[i])
		require.NoError(t, err)

		n, err := conn1To2.Write(msgs[i])
		require.NoError(t, err)
		require.Equal(t, len(msgs[i]), n)
	}

	// Simulate the data being sent form node 1 to 2.
	conn2To1Underlaying.buffer = conn1To2Underlaying.buffer
	conn2To1Underlaying.pos = conn1To2Underlaying.pos
	for i := 0; i < len(msgs); i++ {
		msgRecived := make([]byte, len(msgs[i]))
		var totalRead int
		for totalRead < len(msgs[i]) {
			curRead, err := conn2To1.Read(msgRecived[totalRead:])
			require.NoError(t, err)
			totalRead += curRead
		}

		require.Equal(t, msgs[i], msgRecived)
	}
}

func TestTransformerSingleLargeMessage(t *testing.T) {
	conn1To2Underlaying := &mockedConn{
		buffer: make([]byte, 20*1048),
	}
	conn2To1Underlaying := &mockedConn{}

	conn1To2 := NewConn(conn1To2Underlaying, testTransformerNone, testTransformerNone, time.Second)
	conn2To1 := NewConn(conn2To1Underlaying, testTransformerNone, testTransformerNone, time.Second)

	msgSent := make([]byte, 19*1048)
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
