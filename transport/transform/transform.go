package transform

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/FluffyKebab/pearly/transport"
)

const (
	_maxPacketSize = 10 * 1024
	_lenPacketSize = 4
)

type TransformFunc func([]byte) ([]byte, error)

type Conn struct {
	reader      io.Reader
	writer      io.Writer
	closer      io.Closer
	Transform   TransformFunc
	Detransform TransformFunc

	unread         []byte
	unreadReadPos  int
	unreadWritePos int

	packetOwerflow        []byte
	packetOwerflowWritten int
}

var (
	_ transport.Conn     = &Conn{}
	_ io.ReadWriteCloser = &Conn{}
	_ io.ByteReader      = &Conn{}
)

func NewConn(
	underlayingConn transport.Conn,
	transform TransformFunc,
	detransform TransformFunc,
) *Conn {
	return &Conn{
		reader:         underlayingConn,
		writer:         underlayingConn,
		closer:         underlayingConn,
		Transform:      transform,
		Detransform:    detransform,
		unread:         make([]byte, _maxPacketSize/2),
		packetOwerflow: make([]byte, 0),
	}
}

func (c *Conn) Read(p []byte) (int, error) {
	// If there are bytes left from a previous packet we return those first.
	if c.unreadReadPos < c.unreadWritePos {
		return c.readUnread(p)
	}

	// There are no more unread so we restart the buffer.
	c.unreadReadPos = 0
	c.unreadWritePos = 0

	packetSize, err := c.readAtleastOnePacketIntoUnread()
	if err != nil {
		return 0, err
	}

	transformed := c.unread[_lenPacketSize:(_lenPacketSize + packetSize)]
	if c.Detransform != nil {
		transformed, err = c.Detransform(transformed)
		if err != nil {
			return 0, err
		}
	}

	c.writeUread(transformed)
	return c.readUnread(p)
}

func (c *Conn) readAtleastOnePacketIntoUnread() (int, error) {
	var curPacketRead int
	var err error

	if c.packetOwerflowWritten != 0 {
		curPacketRead = copy(c.unread, c.packetOwerflow[:c.packetOwerflowWritten])
		c.packetOwerflowWritten = 0
	} else {
		curPacketRead, err = c.reader.Read(c.unread)
		if err != nil {
			return 0, err
		}
	}

	if curPacketRead < _lenPacketSize {
		return 0, errors.New("missing packet size")
	}

	curPacketSize := int(binary.LittleEndian.Uint32(c.unread[:_lenPacketSize]))
	if curPacketSize > _maxPacketSize {
		return 0, errors.New("packet size is to large")
	}
	if curPacketSize+_lenPacketSize > len(c.unread) {
		c.unread = append(c.unread, make([]byte, curPacketSize+_lenPacketSize-len(c.unread))...)
	}

	// Read into unread untill the full packet is read.
	for curPacketRead-_lenPacketSize < curPacketSize {
		curRead, err := c.reader.Read(c.unread[curPacketRead:])
		if err != nil {
			return 0, err
		}
		if curRead == 0 {
			return 0, io.ErrNoProgress
		}
		curPacketRead += curRead
	}

	if curPacketRead-_lenPacketSize > curPacketSize {
		c.writeIntoPacketOwerflow(
			c.unread[curPacketSize+_lenPacketSize:],
			curPacketRead-curPacketSize-_lenPacketSize,
		)
	}

	return curPacketSize, nil
}

func (c *Conn) writeIntoPacketOwerflow(data []byte, size int) {
	if len(c.packetOwerflow) < size {
		c.packetOwerflow = append(
			c.packetOwerflow,
			make([]byte, size-len(c.packetOwerflow))...,
		)
	}

	c.packetOwerflowWritten = copy(c.packetOwerflow, data[:size])
	c.packetOwerflowWritten = size
}

func (c *Conn) readUnread(p []byte) (int, error) {
	var n int
	for n < len(p) && c.unreadReadPos < c.unreadWritePos {
		p[n] = c.unread[c.unreadReadPos]
		c.unreadReadPos++
		n++
	}

	return n, nil
}

func (c *Conn) writeUread(p []byte) {
	for i := 0; i < len(p); i++ {
		if c.unreadWritePos >= len(c.unread) {
			c.unread = append(c.unread, p[i])
			continue
		}

		c.unread[c.unreadWritePos] = p[i]
		c.unreadWritePos++
	}
}

func (c *Conn) Write(p []byte) (n int, err error) {
	transformedData := p
	if c.Transform != nil {
		transformedData, err = c.Transform(p)
		if err != nil {
			return 0, err
		}
	}

	// If the size is to large we split the packet in two.
	if len(transformedData) > _maxPacketSize {
		n1, err := c.Write(p[:len(p)/2])
		if err != nil {
			return 0, err
		}
		n2, err := c.Write(p[len(p)/2:])
		return n1 + n2, err
	}

	_, err = c.writer.Write(
		prependInt(transformedData, len(transformedData)),
	)
	return len(p), err
}

func (c *Conn) Close() error {
	return c.closer.Close()
}

func (c *Conn) ReadByte() (byte, error) {
	buf := make([]byte, 1)
	_, err := c.Read(buf)
	return buf[0], err
}

func prependInt(p []byte, n int) []byte {
	p = append(p, make([]byte, _lenPacketSize)...)
	copy(p[_lenPacketSize:], p)
	binary.LittleEndian.PutUint32(p, uint32(n))
	return p
}

func NOPTransform(d []byte) ([]byte, error) {
	return d, nil
}
