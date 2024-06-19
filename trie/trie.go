package trie

import (
	"bytes"
	"cxchain223/crypto/sha3"
	"cxchain223/kvstore"
	"cxchain223/utils/hash"
	"cxchain223/utils/hexutil"
	"cxchain223/utils/rlp"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
)

var EmptyHash = hash.BigToHash(big.NewInt(0))

type ITrie interface {
	Store(key, value []byte) error
	Root() hash.Hash
	Load(key []byte) ([]byte, error)
}

type State struct {
	root *TrieNode
	db   kvstore.KVDatabase
}

type TrieNode struct {
	Path     string
	Children Children
	Leaf     bool
	Value    hash.Hash
}

type Children []Child

type Child struct {
	Path string
	Hash hash.Hash
}

func NewChild(path string, hash hash.Hash) Child {
	return Child{
		Path: path,
		Hash: hash,
	}
}

func (children Children) Len() int {
	return len(children)
}

func (children Children) Less(i, j int) bool {
	return strings.Compare(children[i].Path, children[j].Path) > 0
}

func (children Children) Swap(i, j int) {
	children[i], children[j] = children[j], children[i]
}

func NewState(db kvstore.KVDatabase, root hash.Hash) *State {
	if bytes.Equal(root[:], EmptyHash[:]) {
		return &State{
			db:   db,
			root: NewTrieNode(),
		}
	} else {
		value, err := db.Get(root[:])
		if err != nil {
			panic(err)
		}
		node, err := NodeFromBytes(value)
		if err != nil {
			panic(err)
		}
		return &State{
			db:   db,
			root: node,
		}
	}
}

func NewTrieNode() *TrieNode {
	return &TrieNode{}
}

func NodeFromBytes(data []byte) (*TrieNode, error) {
	var node TrieNode
	err := rlp.DecodeBytes(data, &node)
	return &node, err
}

func (node TrieNode) Bytes() []byte {
	data, _ := rlp.EncodeToBytes(node)
	return data
}

func (node TrieNode) Hash() hash.Hash {
	data := node.Bytes()
	return sha3.Keccak256(data)
}

func (state State) Root() hash.Hash {
	return state.root.Hash()
}

func (state State) LoadTrieNodeByHash(h hash.Hash) (*TrieNode, error) {
	data, _ := state.db.Get(h[:])
	return NodeFromBytes(data)
}

func (state *State) SaveNode(node TrieNode) {
	h := node.Hash()
	state.db.Put(h[:], node.Bytes())
}

func (state State) Load(key []byte) ([]byte, error) {
	path := hexutil.Encode(key)
	paths, hashes := state.FindAncestors(path)
	fmt.Println(paths)
	fmt.Println(hashes)

	matched := strings.Join(paths, "")
	if strings.EqualFold(path, matched) {
		lastHash := hashes[len(hashes)-1]
		leafNode, err := state.LoadTrieNodeByHash(lastHash)
		if err != nil {
			return nil, err
		}
		if !leafNode.Leaf {
			return nil, errors.New("not found")
		}
		return state.db.Get(leafNode.Value[:])
	} else {
		return nil, errors.New("not found")
	}
}

func (state *State) Store(key, value []byte) error {
	valueHash := sha3.Keccak256(value)
	state.db.Put(valueHash[:], value)

	// step 1 find all ancients
	path := hexutil.Encode(key)
	paths, hashes := state.FindAncestors(path)
	prefix := strings.Join(paths, "")
	depth := len(hashes)

	var childPath string
	var childHash hash.Hash
	var node *TrieNode
	if strings.EqualFold(path, prefix) {
		// 已经存在key，直接更新
		leaf, _ := state.LoadTrieNodeByHash(hashes[depth-1])
		leaf.Value = valueHash
		state.SaveNode(*leaf)
		childHash = leaf.Hash()
		childPath = leaf.Path
	} else {
		prefix := strings.Join(paths, "")
		leafPath := path[len(prefix):]
		leafNode := NewTrieNode()
		leafNode.Leaf = true
		leafNode.Path = leafPath
		leafNode.Value = valueHash
		state.SaveNode(*leafNode)
		leafHash := leafNode.Hash()

		node, _ = state.LoadTrieNodeByHash(hashes[depth-1])
		if strings.EqualFold(node.Path, paths[depth-1]) {
			// 插入
			node.Children = append(node.Children, NewChild(leafPath, leafHash))
			sort.Sort(node.Children)
			state.SaveNode(*node)

			childPath = node.Path
			childHash = node.Hash()
		} else {
			// 分叉
			lastMatched := paths[len(paths)-1]
			node.Path = node.Path[len(lastMatched):]
			state.SaveNode(*node)

			newNode := NewTrieNode()
			newNode.Path = lastMatched
			newNode.Children = make(Children, 0)
			newNode.Children = append(newNode.Children, NewChild(leafNode.Path, leafNode.Hash()), NewChild(node.Path, node.Hash()))
			sort.Sort(newNode.Children)

			childHash = newNode.Hash()
			childPath = newNode.Path
		}
	}

	for i := depth - 2; i >= 0; i-- {
		node, _ = state.LoadTrieNodeByHash(hashes[i])
		for _, child := range node.Children {
			if strings.Index(child.Path, childPath) == 0 {
				child.Path = childPath
				child.Hash = childHash
				break
			}
		}

		state.SaveNode(*node)
		childHash = node.Hash()
		childPath = node.Path
	}

	state.root = node

	return nil
}

func (state State) FindAncestors(path string) ([]string, []hash.Hash) {
	current := state.root
	paths, hashes := make([]string, 0), make([]hash.Hash, 0)
	paths = append(paths, "")
	hashes = append(hashes, state.Root())
	prefix := ""
	for {
		flag := false

		for _, child := range current.Children {
			tmp := prefix + child.Path
			length := prefixLength(path, tmp)
			if length == len(tmp) {
				prefix = prefix + child.Path
				paths = append(paths, child.Path)
				hashes = append(hashes, child.Hash)
				flag = true
				data, _ := state.db.Get(child.Hash[:])
				current, _ = NodeFromBytes(data)
				break
			} else if length > len(prefix) {
				l := length - len(prefix)
				str := child.Path[:l]
				paths = append(paths, str)
				hashes = append(hashes, child.Hash)
				return paths, hashes
			}
		}
		if !flag {
			break
		}
	}

	return paths, hashes
}

func prefixLength(s1, s2 string) int {
	length := len(s1)
	if length > len(s2) {
		length = len(s2)
	}
	for i := 0; i < length; i++ {
		if s1[i] != s2[i] {
			return i
		}
	}
	return length
}

// func (state *State) Execute(tx types.Transaction) {
// 	// TODO
// }
