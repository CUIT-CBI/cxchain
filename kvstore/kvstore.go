package kvstore

import "io"

type KVStore interface {
	Put(key, value []byte) error
	Get(key []byte) ([]byte, error)
	Exist(key []byte) (bool, error)
	Delete(key []byte) error
}

type KVDatabase interface {
	KVStore
	io.Closer
}
