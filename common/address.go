package common

import (
	"encoding/hex"
	"log"
	"strings"
)

const (
	// AddressLength is the length of an Ethereum address in bytes.
	AddressLength = 20
)

// Address represents a 20-byte Ethereum address.
type Address [AddressLength]byte

// NewAddress creates a new Address from a byte slice.
func NewAddress(b []byte) Address {
	if len(b) != AddressLength {
		panic("address must be 20 bytes")
	}
	var addr Address
	copy(addr[:], b)
	return addr
}

// Bytes returns the byte representation of the Address.
func BytesToAddress(b []byte) Address {
	var a Address
	if len(b) > len(a) {
		copy(a[:], b[len(b)-20:])
	} else {
		copy(a[20-len(b):], b)
	}
	return a
}

// HexToAddress 将十六进制字符串转换为地址
// 改为大写开头使其成为公共函数，可从其他包访问
func HexToAddress(hexStr string) Address {
	// 移除"0x"前缀
	if strings.HasPrefix(hexStr, "0x") {
		hexStr = hexStr[2:]
	}

	// 解码十六进制字符串
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		log.Fatalf("无法解码地址 %s: %v", hexStr, err)
	}

	// 确保是20字节长
	if len(bytes) != 20 {
		log.Fatalf("地址长度不正确: %d bytes", len(bytes))
	}

	return BytesToAddress(bytes)
}
