package mpt

type Nibbles []byte

// 将[]byte转为nibbles（半字节数组）
func BytesToNibbles(bytes []byte) []byte {
    nibbles := make([]byte, len(bytes)*2)
    for i, b := range bytes {
        nibbles[2*i] = b >> 4       // 高4位
        nibbles[2*i+1] = b & 0x0F   // 低4位
    }
    return nibbles
}