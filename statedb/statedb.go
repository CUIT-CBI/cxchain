package statedb

import (
	"chain/common"
	"chain/trie/mpt"
	"fmt"
)

type statedb struct {
	trie *mpt.MPT
}

// NewStateDB creates a new StateDB instance with the provided MPT.
func NewStateDB(trie *mpt.MPT) StateDB {
	return &statedb{
		trie: trie,
	}
}

// 通过地址从 MPT 中获取账户
func (s statedb) Get(address common.Address) (*common.Account, error) {
	return s.trie.Get(address[:])
}

// 通过地址和账户对象将账户存储到 MPT 中
func (s statedb) Put(address common.Address, account *common.Account) error {
	key := address[:]
	value := account.Bytes() //将账户对象转换为json字节数组
	return s.trie.Insert(key, value)
}

// Root returns the root hash of the MPT.
func (s *statedb) Root() (common.Hash, error) {
	if s.trie == nil || s.trie.Root == nil {
		return common.Hash{}, fmt.Errorf("MPT 树为空")
	}

	// 假设有某种方式可以获取根节点的哈希
	rootHash  := s.trie.StoreNode(s.trie.Root)

	return rootHash, nil
}

func (s *statedb) SetStateRoot(root common.Hash) error{
	
	fmt.Printf("MPT 根地址已更新为: %x\n", root)
	return  s.trie.SetRoot(root)
}
