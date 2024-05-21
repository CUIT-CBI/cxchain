package trie

import (
	"cxchain223/utils/hash"
	"math/big"
)

var EmptyHash = hash.BigToHash(big.NewInt(0))

type ITrie interface {
	Store(key, value []byte) error
	Root() hash.Hash
	Load(key []byte) ([]byte, error)
}

// TODO 实现ITrie
