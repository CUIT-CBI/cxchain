package main

import (
	"cxchain223/kvstore"
	"cxchain223/trie"
	"fmt"
)

func main() {
	db := kvstore.NewLevelDB("./testdb")
	state := trie.NewState(db, trie.EmptyHash)
	state.Store([]byte("apple"), []byte("apple"))
	state.Store([]byte("apply"), []byte("apply"))
	state.Store([]byte("application"), []byte("application"))
	state.Store([]byte("banana"), []byte("banana"))
	state.Store([]byte("band"), []byte("band"))
	value, err := state.Load([]byte("apple"))
	fmt.Println(string(value), err)

}
