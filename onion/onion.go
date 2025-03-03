package onion

import (
	"time"
)

type RelayRequest struct {
	NextNodeAddr string
	WaitTime     time.Duration
	NextNodeData []byte
}

type ExistRequest []byte

type ExistRequestHandler func(*ExistRequest) error

type Pealer interface {
	Peal([]byte) (*RelayRequest, *ExistRequest, error)
}

type Wrapper interface {
	WrapRelayRequest(RelayRequest) ([]byte, error)
	WrapExistRequest(ExistRequest) ([]byte, error)
}
