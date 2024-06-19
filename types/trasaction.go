package types

import (
	"cxchain221/crypto/secp256k1"
	"cxchain221/crypto/sha3"
	"cxchain221/utils/rlp"
	"hash"
	"math/big"
)

type Transaction struct {
	txdata
	sig
}

type Receipt struct {
	TxHash hash.Hash
	Status int
	// log
}

type txdata struct {
	To       Address
	Value    uint64
	Nonce    uint64
	Gas      uint64
	GasPrice uint64
	Input    []byte
}

type sig struct {
	R, S *big.Int
	V    uint8
}

func (tx Transaction) From() Address {
	txdata := tx.txdata
	toSign, _ := rlp.EncodeToBytes(txdata)
	msg := sha3.Keccak256(toSign)
	sig := make([]byte, 65)
	// 把RSV写进去
	pubKey, err := secp256k1.RecoverPubkey(msg, sig)
	if err != nil {
		// TODO
		// return
	}
	return PubKeyToAddress(pubKey)
}
