package types

type Address [20]byte

func PubKeyToAddress(pub []byte) Address {
	// h := sha3.Keccak256(pub)
	var addr Address
	// TODO hash得到addr
	return addr
}
