package encrypted

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/gob"
	"io"
	mrand "math/rand"
	"strings"
	"sync"

	"github.com/FluffyKebab/pearly/transport"
)

type Conn struct {
	lock        *sync.Mutex
	conn        transport.Conn
	peerPubKey  *rsa.PublicKey
	nodePrivKey *rsa.PrivateKey
	decoder     *gob.Decoder
	encoder     *gob.Encoder

	ID             int
	remoteID       []byte
	unread         []byte
	unreadReadPos  int
	unreadWritePos int
	remotePort     string
}

var (
	_ transport.Conn            = &Conn{}
	_ transport.RemoteAddrHaver = &Conn{}
	_ transport.RemoteIDHaver   = &Conn{}
	_ io.ReadWriteCloser        = &Conn{}
	_ io.ByteReader             = &Conn{}
)

func NewConn(
	underlayingConn transport.Conn,
	peerPubKey *rsa.PublicKey,
	nodePrivateKey *rsa.PrivateKey,
	gobDecoder *gob.Decoder,
	gobEncoder *gob.Encoder,
	peerID []byte,
	remotePort string,
) *Conn {
	return &Conn{
		lock:        &sync.Mutex{},
		ID:          mrand.Int(),
		remoteID:    peerID,
		conn:        underlayingConn,
		unread:      make([]byte, 1048),
		peerPubKey:  peerPubKey,
		nodePrivKey: nodePrivateKey,
		decoder:     gobDecoder,
		encoder:     gobEncoder,
		remotePort:  remotePort,
	}
}

func (c *Conn) Read(p []byte) (n int, err error) {
	// If there are bytes left from a previous packet we return those first.
	if c.unreadReadPos < c.unreadWritePos {
		return c.readUnread(p)
	}

	// There are no more unread so we restart the buffer.
	c.unreadReadPos = 0
	c.unreadWritePos = 0

	var pckt packet
	err = c.decoder.Decode(&pckt)
	if err != nil {
		return 0, err
	}

	for _, msg := range pckt.Data {
		plaintext, err := c.nodePrivKey.Decrypt(rand.Reader, msg, nil)
		if err != nil {
			return 0, err
		}
		c.writeUread(plaintext)
	}

	return c.readUnread(p)
}

func (c *Conn) Write(p []byte) (n int, err error) {
	bachedData := bactchData(p, c.peerPubKey.Size()-11)
	packet := packet{Data: make([][]byte, 0, len(bachedData))}

	for _, msg := range bachedData {
		ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, c.peerPubKey, msg)
		if err != nil {
			return 0, err
		}

		packet.Data = append(packet.Data, ciphertext)
	}

	err = c.encoder.Encode(packet)
	return len(p), err
}

func (c *Conn) RemoteAddr() string {
	if remoteHaver, ok := c.conn.(transport.RemoteAddrHaver); ok {
		remoteAddr := remoteHaver.RemoteAddr()
		splitRemoteAddr := strings.Split(remoteAddr, ":")
		if len(splitRemoteAddr) != 2 {
			return remoteAddr
		}
		return strings.Join([]string{splitRemoteAddr[0], c.remotePort}, ":")
	}
	return ""
}

func (c *Conn) RemoteID() []byte {
	return c.remoteID
}

func (c *Conn) Close() error {
	return c.conn.Close()
}

func (c *Conn) ReadByte() (byte, error) {
	buf := make([]byte, 1)
	_, err := c.Read(buf)
	return buf[0], err
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

type packet struct {
	Data [][]byte
}

func bactchData(data []byte, packetSize int) [][]byte {
	res := make([][]byte, 0, len(data)/packetSize)
	for i := 0; i < len(data); i += packetSize {
		res = append(res, data[i:min(len(data), i+packetSize)])
	}

	return res
}
