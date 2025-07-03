package mpt
import (
	"golang.org/x/crypto/sha3"
)

func SHA3_256(data []byte) [32]byte {
	return sha3.Sum256(data)
}