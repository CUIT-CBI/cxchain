package trie

import "chain/common"


type Trie interface{
	//search finds the value associated with the given key.
	Search(key []byte) ([]byte, error)
	//Insert adds a key-value pair to the trie.
	Insert(key, value []byte) error
	//returns the root hash of the trie.
	Root() common.Hash
}

