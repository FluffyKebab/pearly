package tcp

import (
	"net"

	"github.com/FluffyKebab/pearly/transport"
)

type connection struct {
	conn net.Conn
}

var (
	_ transport.Conn            = connection{}
	_ transport.RemoteAddrHaver = connection{}
)

func (c connection) Read(p []byte) (n int, err error) {
	n, err = c.conn.Read(p)
	return n, err
}

func (c connection) Write(p []byte) (n int, err error) {
	n, err = c.conn.Write(p)
	return n, err
}

func (c connection) Close() error {
	return c.conn.Close()
}

func (c connection) RemoteAddr() string {
	if c.conn.RemoteAddr() == nil {
		return ""
	}

	return c.conn.RemoteAddr().String()
}
