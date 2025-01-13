package storage

import "errors"

var ErrNotFound = errors.New("value not found")

type Hashtable interface {
	Get(key []byte) ([]byte, error)
	Set(key, value []byte) error
}

type hashtable struct {
	data map[string][]byte
}

var _ Hashtable = hashtable{}

func NewHashtable() hashtable {
	return hashtable{
		data: make(map[string][]byte),
	}
}

func (h hashtable) Get(key []byte) ([]byte, error) {
	res, ok := h.data[string(key)]
	if !ok {
		return nil, ErrNotFound
	}
	return res, nil
}

func (h hashtable) Set(key, value []byte) error {
	h.data[string(key)] = value
	return nil
}
