package onion

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	_maxNextNodeAddrSize = 256
)

type Request struct {
	// NextNodeAddr is the contact information for the next node in the circut.
	NextNodeAddr string

	// MaxRandomWaitTime is the maximum time this noe will wait to relay from
	// the next node in the circut to the previous one and vice versa.
	MaxRandomWaitTime time.Duration
}

func (r *Request) UnmarshalBinary(data []byte) error {
	if len(data) != _maxNextNodeAddrSize+8 {
		return errors.New("invalid data size")
	}

	r.MaxRandomWaitTime = time.Duration(readInt64(data[0:8]))
	r.NextNodeAddr = strings.TrimRight(string(data[8:]), " ")
}

func (r *Request) MarshalBinary() ([]byte, error) {

}

// readRequest reads the request from a pealed connection without consuming
// any data not related to the request.
func readRequest(r io.Reader) (*Request, error) {
	requestSize := _maxNextNodeAddrSize + 8
	buf := make([]byte, requestSize)

	n, err := r.Read(buf)
	if err != nil {
		return nil, err
	}
	if n != requestSize {
		return nil, fmt.Errorf("%w: request need to be sent at same time", ErrInvalidRequest)
	}

	req := &Request{}
	return req, req.UnmarshalBinary(buf)
}

func writeRequest(w io.Writer, req *Request) error {
	data, err := req.MarshalBinary()
	if err != nil {
		return err
	}

	_, err = w.Write(data)
	return err
}
