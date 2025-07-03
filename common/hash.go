package common

import "golang.org/x/crypto/sha3"

const (
	// HashLength is the length of a hash in bytes.
	HashLength = 32
)

type Hash [HashLength]byte

var EmptyHash Hash = [HashLength]byte{}

// Keccak256 计算输入的Keccak-256哈希
func Keccak256(data ...[]byte) []byte {
	d := sha3.NewLegacyKeccak256()
	for _, b := range data {
		d.Write(b)
	}
	return d.Sum(nil)
}

// Keccak256Hash 计算并返回输入的Keccak-256哈希
func Keccak256Hash(data ...[]byte) Hash {
	return BytesToHash(Keccak256(data...))
}

// BytesToHash 将字节转换为哈希
// 此函数需要正确处理输入长度
func BytesToHash(b []byte) Hash {
	var h Hash

	// 检查输入长度
	if len(b) == 0 {
		return h
	}

	// 如果输入太长，截断
	if len(b) > len(h) {
		b = b[:len(h)]
	}

	// 如果输入太短，填充
	copy(h[len(h)-len(b):], b)

	return h
}
