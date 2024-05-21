package sha3

import (
	"golang.org/x/crypto/sha3"

	"cxchain223/utils/hash"
)

func Sha3(value []byte) hash.Hash {
	sha := sha3.NewLegacyKeccak256()
	return hash.BytesToHash(sha.Sum(value))
}
