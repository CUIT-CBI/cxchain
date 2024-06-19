package types

import "hash"

type Account struct {
	Amount uint64
	Nonce  uint64

	CodeHash hash.Hash
	Root     hash.Hash
}
