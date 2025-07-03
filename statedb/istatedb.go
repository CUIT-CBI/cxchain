package statedb

import "chain/common"

type StateDB interface {
	Get(address common.Address) (*common.Account, error)
	Put(address common.Address, account *common.Account) error
	Root() (common.Hash, error) // 获取 MPT 的根地址
	SetStateRoot(root common.Hash) error//更新MPT的根地址
}