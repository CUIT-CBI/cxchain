package mpt

import (
	"bytes"
	"chain/common"
	"encoding/gob"
	"encoding/json"
	"fmt"
)

func init() {
	gob.Register(&LeafNode{})
	gob.Register(&ExtensionNode{})
	gob.Register(&BranchNode{})
}

// Node 接口定义
type Node interface {
	GetChildren() ([]Node, error)
	GetValue() ([]byte, error)
	GetPath() ([]byte, error)
	GetType() string
	GetHash() common.Hash
	Serialize() []byte
}

// === LeafNode ===

type LeafNode struct {
	Path  []byte
	Value []byte
}


func (n *LeafNode) GetChildren() ([]Node, error) {
	return nil, nil
}

func (n *LeafNode) GetValue() ([]byte, error) {
	return n.Value, nil
}

func (n *LeafNode) GetPath() ([]byte, error) {
	return n.Path, nil
}

func (n *LeafNode) GetType() string {
	return "leaf"
}

func (n *LeafNode) Serialize() ([]byte) {
	data,_ := json.Marshal(n)
	return data
}

func (n *LeafNode) GetHash() common.Hash {
	return SHA3_256(n.Serialize())
}

// === ExtensionNode ===

type ExtensionNode struct {
	Path  []byte
	Child Node
}



func (n *ExtensionNode) GetChildren() ([]Node, error) {
	return []Node{n.Child}, nil
}

func (n *ExtensionNode) GetValue() ([]byte, error) {
	return nil, nil
}

func (n *ExtensionNode) GetPath() ([]byte, error) {
	return n.Path, nil
}

func (n *ExtensionNode) GetType() string {
	return "extension"
}

func (n *ExtensionNode) Serialize() []byte {
	data,_ := json.Marshal(n)
	return data
}
func (n *ExtensionNode) GetHash()common.Hash {
	return SHA3_256(n.Serialize())
}

// === BranchNode ===

type BranchNode struct {
	Children [16]Node
	Value    []byte
}

func (n *BranchNode) GetChildren() ([]Node, error) {
    children := make([]Node, 16)
    for i := 0; i < 16; i++ {
        children[i] = n.Children[i] // 不管是否为nil，都保留
    }
    return children, nil
}


func (n *BranchNode) GetValue() ([]byte, error) {
	return n.Value, nil
}

func (n *BranchNode) GetPath() ([]byte, error) {
	return nil, nil // branch node has no path
}

func (n *BranchNode) GetType() string {
	return "branch"
}

func (n *BranchNode) Serialize() []byte {
	data, _ := json.Marshal(n)
	return data
}
func (n *BranchNode) GetHash() common.Hash {
	return SHA3_256(n.Serialize())

}

// === 编码和解码节点 ===

func EncodeNode(node Node) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	if err := enc.Encode(node.GetType()); err != nil {
		return nil, err
	}

	switch n := node.(type) {
	case *LeafNode:
		err := enc.Encode(n)
		return buf.Bytes(), err
	case *ExtensionNode:
		childData, err := EncodeNode(n.Child)
		if err != nil {
			return nil, err
		}
		if err := enc.Encode(n.Path); err != nil {
			return nil, err
		}
		if err := enc.Encode(childData); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	case *BranchNode:
		err := enc.Encode(n)
		return buf.Bytes(), err
	default:
		return nil, fmt.Errorf("unknown node type: %T", n)
	}
}

func DecodeNode(data []byte) (Node, error) {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)

	var nodeType string
	if err := dec.Decode(&nodeType); err != nil {
		return nil, err
	}

	switch nodeType {
	case "leaf":
		var node LeafNode
		if err := dec.Decode(&node); err != nil {
			return nil, err
		}
		return &node, nil
	case "branch":
		var node BranchNode
		if err := dec.Decode(&node); err != nil {
			return nil, err
		}
		return &node, nil
	case "extension":
		var path []byte
		var childData []byte
		if err := dec.Decode(&path); err != nil {
			return nil, err
		}
		if err := dec.Decode(&childData); err != nil {
			return nil, err
		}
		childNode, err := DecodeNode(childData)
		if err != nil {
			return nil, err
		}
		return &ExtensionNode{Path: path, Child: childNode}, nil
	default:
		return nil, fmt.Errorf("unknown node type: %s", nodeType)
	}
}

