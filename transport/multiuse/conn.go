package multiuse

import (
	"io"
	"net"
	"sync"

	"github.com/FluffyKebab/pearly/transport"
)

type Conn struct {
	connMuxer *ConnMuxer

	id               string
	closed           bool
	unreadChunks     [][]byte
	unreadChunkPos   int
	unreadInChunkPos int

	lock              *sync.Mutex
	isWaiting         bool
	waitingDataSender chan struct{}
}

func NewConn(connMuxer *ConnMuxer, id string) *Conn {
	return &Conn{
		connMuxer:         connMuxer,
		id:                id,
		unreadChunks:      make([][]byte, 0),
		waitingDataSender: make(chan struct{}),
		lock:              &sync.Mutex{},
	}
}

func (c *Conn) writeUread(data []byte, closing bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.unreadChunks = append(c.unreadChunks, data)
	c.closed = closing

	if c.isWaiting {
		c.waitingDataSender <- struct{}{}
	}
}

func (c *Conn) Read(p []byte) (n int, err error) {
	if len(c.unreadChunks) == 0 && c.closed {
		return 0, io.EOF
	}

	// If there is no data, wait until there is or the peer closed.
	c.lock.Lock()
	if len(c.unreadChunks) == c.unreadChunkPos {
		c.isWaiting = true
		c.lock.Unlock()

		<-c.waitingDataSender
		if c.closed {
			return 0, io.EOF
		}
		c.isWaiting = false
	} else {
		c.lock.Unlock()
	}

	for c.unreadChunkPos < len(c.unreadChunks) {
		for c.unreadInChunkPos < len(c.unreadChunks[c.unreadChunkPos]) {
			numCopied := copy(p[n:], c.unreadChunks[c.unreadChunkPos][c.unreadInChunkPos:])
			n += numCopied
			c.unreadInChunkPos += numCopied

			if n >= len(p) {
				return n, nil
			}
		}

		c.unreadChunkPos++
		c.unreadInChunkPos = 0
	}

	// We have written all of the buffer so we can reset it.
	c.unreadChunks = make([][]byte, 0)
	c.unreadChunkPos = 0
	c.unreadInChunkPos = 0
	return n, nil
}

func (c *Conn) Write(p []byte) (n int, err error) {
	if c.closed {
		return 0, net.ErrClosed
	}

	return len(p), c.connMuxer.WriteToConnWithID(c.id, p)
}

func (c *Conn) RemoteAddr() string {
	if addr, ok := c.connMuxer.underlayingConn.(transport.RemoteAddrHaver); ok {
		return addr.RemoteAddr()
	}
	return ""
}

func (c *Conn) RemoteID() []byte {
	return c.connMuxer.peerID
}

func (c *Conn) Close() error {
	return c.connMuxer.CloseConnWithID(c.id)
}

func (c *Conn) ReadByte() (byte, error) {
	buf := make([]byte, 1)
	_, err := c.Read(buf)
	return buf[0], err
}
