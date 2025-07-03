package common

import(
	"encoding/json"
)

type Account struct {
	Address Address `json:"address"` // Account address
	Balance uint64  `json:"balance"` // Account balance in wei
	Nonce   uint64  `json:"nonce"`   // Account nonce, used to prevent replay attacks
}

// serialize the account to bytes
func(a *Account) Bytes() []byte{
	//Use Json seriaization to convert the account to bytes
	data, err := json.Marshal(a)
	if err != nil {
		panic("failed to marshal account to bytes: " + err.Error())
	}
	return data
}

// BytesToAccount converts bytes to an Account.
func BytesToAccount(data []byte) *Account {
	var acc Account
	err := json.Unmarshal(data, &acc)
	if err != nil {
		panic("failed to unmarshal bytes to account: " + err.Error())
	}
	return &acc
}

