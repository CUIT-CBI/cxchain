package types

import (
	"cxchain223/crypto/secp256k1"
	"cxchain223/crypto/sha3"
	"cxchain223/utils/hexutil"
	"cxchain223/utils/rlp"
	"fmt"
	"hash"
	"math/big"
)

type Receiption struct {
	TxHash hash.Hash
	Status int
	// GasUsed int
	// Logs
}

type Transaction struct {
	txdata
	signature
}

type txdata struct {
	To       Address
	Nonce    uint64
	Value    uint64
	Gas      uint64
	GasPrice uint64
	Input    []byte
}

type signature struct {
	R, S *big.Int
	V    uint8
}

func (tx Transaction) From() Address {
	txdata := tx.txdata
	toSign, err := rlp.EncodeToBytes(txdata)
	fmt.Println(hexutil.Encode(toSign), err)
	msg := sha3.Keccak256(toSign)
	sig := make([]byte, 65)
	pubKey, err := secp256k1.RecoverPubkey(msg[:], sig)
	if err != nil {
		// TODO 返回一个空地址
		panic(err)
	}
	return PubKeyToAddress(pubKey)
}
