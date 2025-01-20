package storage

import (
	"crypto/sha256"
	"errors"
)

var ErrNotFound = errors.New("value not found")

type Hashtable interface {
	Get(key []byte) ([]byte, error)
	Set(key, value []byte) error
}

type Hasher interface {
	Hash(value []byte) ([]byte, error)
}

type hashtable struct {
	data map[string][]byte
}

var _ Hashtable = &hashtable{}

func NewHashtable() *hashtable {
	return &hashtable{
		data: make(map[string][]byte),
	}
}

func (h *hashtable) Get(key []byte) ([]byte, error) {
	res, ok := h.data[string(key)]
	if !ok {
		return nil, ErrNotFound
	}
	return res, nil
}

func (h *hashtable) Set(key, value []byte) error {
	h.data[string(key)] = value

	return nil
}

type sha256Hasher struct{}

var _ Hasher = sha256Hasher{}

func NewHasher() sha256Hasher {
	return sha256Hasher{}
}

func (h sha256Hasher) Hash(value []byte) ([]byte, error) {
	hash := sha256.New()
	_, err := hash.Write(value)
	return hash.Sum(nil), err
}
