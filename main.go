package main

import (
	"cxchain223/kvstore"
	"cxchain223/trie"
	"fmt"
)

func main() {
	db := kvstore.NewLevelDB("./leveldb")
	state := trie.NewState(db, trie.EmptyHash)
	state.Store([]byte("hello"), []byte("world"))
	value, err := state.Load([]byte("hello"))
	fmt.Println(string(value), err)
}
