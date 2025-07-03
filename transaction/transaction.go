package transaction

import (
	"chain/common"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// 🌟 去掉hash字段，保持结构简洁
type Transaction struct {
	Txdata
	Signature []byte
	// 🌟 删除hash字段
}

type Txdata struct {
	To       common.Address
	GasPrice uint64
	Nonce    uint64
	ChainID  *big.Int
	GasLimit uint64
	Value    *big.Int
	Data     []byte
}

type signature struct {
	R, S, V *big.Int
}

// 从Signature字节数组中提取V,R,S
func (tx *Transaction) extractSignature() (v, r, s *big.Int) {
	if tx.Signature == nil || len(tx.Signature) != 65 {
		return big.NewInt(0), big.NewInt(0), big.NewInt(0)
	}

	r = new(big.Int).SetBytes(tx.Signature[:32])
	s = new(big.Int).SetBytes(tx.Signature[32:64])
	v = new(big.Int).SetUint64(uint64(tx.Signature[64] + 27))

	// 应用EIP-155
	if tx.ChainID != nil && tx.ChainID.Sign() != 0 {
		v = new(big.Int).Add(
			new(big.Int).SetUint64(uint64(tx.Signature[64])),
			new(big.Int).Add(
				big.NewInt(35),
				new(big.Int).Mul(tx.ChainID, big.NewInt(2)),
			),
		)
	}

	return v, r, s
}

// Serialize 序列化交易数据（不包含签名）
func (tx *Transaction) Serialize() ([]byte, error) {
	if tx.ChainID == nil {
		tx.ChainID = big.NewInt(0)
	}
	if tx.Value == nil {
		tx.Value = big.NewInt(0)
	}

	data := []interface{}{
		tx.Nonce,
		tx.GasPrice,
		tx.GasLimit,
		tx.To,
		tx.Value,
		tx.Data,
	}

	return rlp.EncodeToBytes(data)
}

// 🌟 简化GetHash：每次都直接计算，不缓存
func (tx *Transaction) GetHash() common.Hash {
	// 序列化交易数据
	data, err := tx.Serialize()
	if err != nil {
		return common.Hash{}
	}

	// 直接计算并返回哈希
	return common.Keccak256Hash(data)
}

// SetSignature 使用私钥对交易进行签名
func (tx *Transaction) SetSignature(privateKey *ecdsa.PrivateKey) []byte {
	// 每次都重新计算哈希
	hash := tx.GetHash()

	signature, err := crypto.Sign(hash[:], privateKey)
	if err != nil {
		return []byte{}
	}

	tx.Signature = signature
	return signature
}

// From 返回交易发送者的地址
func (tx *Transaction) From() common.Address {
	if tx.Signature == nil {
		return common.Address{}
	}

	// 每次都重新计算哈希
	hash := tx.GetHash()

	v, r, s := tx.extractSignature()

	addr, err := RecoverAddress(hash[:], v, r, s, tx.ChainID)
	if err != nil {
		return common.Address{}
	}

	return addr
}

// RecoverAddress 从哈希和签名组件中恢复发送者地址
func RecoverAddress(hash []byte, v, r, s *big.Int, chainID *big.Int) (common.Address, error) {
	sig := make([]byte, 65)

	rBytes := r.Bytes()
	copy(sig[32-len(rBytes):32], rBytes)

	sBytes := s.Bytes()
	copy(sig[64-len(sBytes):64], sBytes)

	var recoveryID byte
	vInt := v.Uint64()

	if chainID != nil && chainID.Sign() != 0 {
		recoveryID = byte((vInt - 35 - chainID.Uint64()*2) % 2)
	} else if vInt == 27 || vInt == 28 {
		recoveryID = byte(vInt - 27)
	} else {
		return common.Address{}, fmt.Errorf("无效的v值: %v", v)
	}

	sig[64] = recoveryID

	pubKey, err := crypto.SigToPub(hash, sig)
	if err != nil {
		return common.Address{}, err
	}

	ethAddr := crypto.PubkeyToAddress(*pubKey)
	return common.BytesToAddress(ethAddr.Bytes()), nil
}

// NewTransaction 构造函数
func (tx *Transaction) NewTransaction(to common.Address, gasPrice uint64, nonce uint64) *Transaction {
	return &Transaction{
		Txdata: Txdata{
			To:       to,
			GasPrice: gasPrice,
			Nonce:    nonce,
			ChainID:  big.NewInt(1),
			GasLimit: 21000,
			Value:    big.NewInt(0),
			Data:     []byte{},
		},
		Signature: nil,
		// 🌟 不需要初始化hash字段
	}
}

// 辅助结构
type Txbox struct {
	Tx       []Transaction
	GasPrice uint64
}

type Boxes []Txbox

func (b *Boxes) Len() int {
	return len(*b)
}

func (b *Boxes) Less(i, j int) bool {
	return (*b)[i].GasPrice > (*b)[j].GasPrice
}

func (b *Boxes) Swap(i, j int) {
	(*b)[i], (*b)[j] = (*b)[j], (*b)[i]
}
