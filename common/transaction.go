package common

import (
	"crypto/sha3"
	"math/big"
)

type Transaction struct {
	txdata
	signature
}

type txdata struct {
	Nonce    uint64
	GasPrice uint64
	GasLimit uint64
	To       Address
	Value    *big.Int
	Input    []byte
}

type signature struct {
	R, S *big.Int
	V    uint8
}

func (tx *Transaction) From() Address {
	hash := tx.Hash()
	// secp256k1 uses the last byte of V to determine the recovery ID.
	return RecoverAddress(hash, tx.R, tx.S, tx.V)
}

func RecoverAddress(hash []byte, r, s *big.Int, v uint8) Address {
	// TODO: Implement the logic to recover the address from the signature.
	// This function should implement the logic to recover the address from the signature.
	// For simplicity, we assume it returns a dummy address here.
	// In a real implementation, you would use a cryptographic library to perform this operation.
	return Address{} // Replace with actual recovery logic
}

func (tx *Transaction) Hash() []byte {
	// sha3 hash of the transaction data
	hash := sha3.NewLegacyKeccak256()
	hash.Write(tx.txdata)
	return hash.Sum(nil)
}
