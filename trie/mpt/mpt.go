package mpt

import (
	"bytes"
	"chain/common"
	"chain/kvstore"
	"errors"
	"fmt"
)

type MPT struct {
	Root Node
	DB   kvstore.KVStore
}

func NewMPT(db kvstore.KVStore, root common.Hash) (*MPT, error) {
	mpt := &MPT{DB: db}
	// 空根哈希（全零），表示新建空Trie
	if root == (common.Hash{}) {
		// 新建一个空节点作为根，BranchNode or nil
		mpt.Root = nil
		return mpt, nil
	}

	// 非空根哈希，从数据库加载根节点
	rootNode, err := mpt.LoadNode(root)
	if err != nil {
		return nil, err
	}
	mpt.Root = rootNode
	return mpt, nil
}

// use the hash load the node from the database
func (m *MPT) LoadNode(hash common.Hash) (Node, error) {
	if hash == common.EmptyHash {
		return nil, nil
	}
	data, err := m.DB.Get(hash[:])
	if err != nil {
		return nil, err
	}
	node, err := DecodeNode(data)
	if err != nil {
		return nil, err
	}
	return node, nil
}

// StoreNode stores the node in the database and returns its hash.
func (m *MPT) StoreNode(node Node) common.Hash {
	data, err := EncodeNode(node)
	if err != nil {
		fmt.Printf("❌ EncodeNode 失败: %v\n", err)
		return common.EmptyHash
	}
	hash := SHA3_256(data)
	err = m.DB.Put(hash[:], data)
	if err != nil {
		fmt.Printf("❌ 数据库存储失败: %v\n", err)
		return common.EmptyHash
	}
	fmt.Println("📦 存储节点类型:", node.GetType())
	return hash
}

// 计算两个字节切片 key 和 path 的最长公共前缀长度
func (m *MPT) prefixLength(key, path []byte) int {
	if len(key) == 0 || len(path) == 0 {
		return 0
	}
	minLen := len(key)
	if len(path) < minLen {
		minLen = len(path)
	}
	for i := 0; i < minLen; i++ {
		if key[i] != path[i] {
			return i
		}
	}
	return minLen
}

// 根据给定的 key，沿着 MPT（Merkle Patricia Trie）查找最长匹配的路径
// 同时返回经过的路径（节点路径的编码）和节点集合。
func (m *MPT) FindLongestPrefix(key []byte) (path [][]byte, nodes []Node, err error) {
	// 从 MPT 的根节点开始查找
	current := m.Root
	leftKey := key

	fmt.Printf("🔎 开始查找路径，key: %x\n", key)

	if current == nil {
		fmt.Println("MPT 为空")
		return nil, nil, nil
	}

	// 当 current 节点不为空且 leftKey 还有剩余时，继续查找
	for current != nil && len(leftKey) >= 0 {
		nodeType := current.GetType()
		nodePath, e := current.GetPath()

		if e != nil {
			err = e
			return
		}

		fmt.Printf("👉 当前节点: 类型=%s, 路径=%x\n", nodeType, nodePath)
		fmt.Printf("📊 剩余key: %x\n", leftKey)

		// 计算前缀匹配长度
		prefixLen := m.prefixLength(leftKey, nodePath)
		fmt.Printf("📏 前缀匹配长度: %d (节点路径长度: %d)\n", prefixLen, len(nodePath))

		// 记录经过的节点和对应路径
		nodes = append(nodes, current)
		path = append(path, nodePath)

		// 如果前缀长度小于节点路径长度，说明不完全匹配
		if prefixLen < len(nodePath) {
			fmt.Printf("🛑 部分匹配，终止查找\n")
			break
		}

		// 剩余 key 去掉匹配过的部分，准备往下一层继续匹配
		leftKey = leftKey[prefixLen:]
		fmt.Printf("➡️ 匹配后剩余key: %x\n", leftKey)

		// 根据当前节点类型决定如何继续查找
		switch nodeType {
		case "leaf":
			// 遇到叶子节点，最长匹配结束，返回路径和节点
			fmt.Printf("🍃 找到叶子节点，路径: %x\n", nodePath)
			return path, nodes, nil

		case "extension":
			// 扩展节点只有一个子节点，继续向下查找
			children, e := current.GetChildren()
			if e != nil {
				err = e
				return
			}

			// 获取子节点
			childNode := children[0]
			if childNode == nil {
				fmt.Printf("⚠️ 扩展节点的子节点为空\n")
				return path, nodes, nil
			}

			childType := childNode.GetType()
			fmt.Printf("👶 子节点类型: %s\n", childType)

			current = childNode

		case "branch":
			// 处理分支节点
			// 如果剩余 key 为空，直接返回分支节点
			if len(leftKey) == 0 {
				fmt.Printf("🔄 剩余键为空，返回分支节点\n")
				return path, nodes, nil
			}

			children, e := current.GetChildren()
			// 获取分支节点的子节点
			if e != nil {
				err = e
				return
			}

			// 获取索引和对应的子节点
			index := leftKey[0]

			// 检查索引是否有效
			if int(index) >= len(children) {
				fmt.Printf("⚠️ 分支索引 %d 超出范围\n", index)
				return path, nodes, nil
			}

			// 检查子节点是否存在
			childNode := children[index]
			if childNode == nil {
				fmt.Printf("⚠️ 分支索引 %d 处子节点为空\n", index)
				return path, nodes, nil
			}

			// 添加分支索引到路径
			path = append(path, []byte{index})
			// 更新当前节点为子节点
			current = childNode
			// 去掉第一个字节，继续查找
			leftKey = leftKey[1:]

		default:
			fmt.Printf("❓ 未知节点类型: %s\n", nodeType)
			return path, nodes, nil
		}

	}

	fmt.Printf("🏁 查找完成，返回 %d 个节点\n", len(nodes))
	return path, nodes, nil
}

func (m *MPT) Get(key []byte) (*common.Account, error) {
	if m.Root == nil {
		return nil, fmt.Errorf("MPT 为空")
	}

	nibbles := BytesToNibbles(key)
	fmt.Printf("[Get] 查询的 key 转为 nibbles: %v\n", nibbles)

	node := m.Root

	for {
		switch n := node.(type) {
		//如果是分支节点，就根据 nibbles 的第一个字节选择子节点
		case *BranchNode:
			fmt.Printf("[Get] 当前节点类型：BranchNode\n")

			if len(nibbles) == 0 {
				// 如果 nibbles 为空，直接返回 BranchNode 的值
				if n.Value == nil {
					return nil, fmt.Errorf("账户不存在（BranchNode 的值为空）")
				}
				accountBytes, err := m.DB.Get(n.Value)
				if err != nil {
					return nil, fmt.Errorf("数据库中找不到账户数据，哈希：%x", n.Value)
				}
				return common.BytesToAccount(accountBytes), nil
			}
			// 如果 nibbles 不为空，选择对应的子节点
			child := n.Children[nibbles[0]]
			if child == nil {
				return nil, fmt.Errorf("账户不存在（BranchNode 子节点为空），查找的 nibble: %x", nibbles[0])
			}
			node = child
			nibbles = nibbles[1:]
			fmt.Printf("[Get] 匹配 BranchNode，剩余 nibbles：%v\n", nibbles)

		case *ExtensionNode:
			fmt.Printf("[Get] 当前节点类型：ExtensionNode，路径：%v\n", n.Path)

			// 如果 nibbles 的长度小于 ExtensionNode 的路径长度，说明不匹配
			if len(nibbles) < len(n.Path) || !bytes.Equal(n.Path, nibbles[:len(n.Path)]) {
				fmt.Printf("[Get] ExtensionNode 路径不匹配，nibbles: %v, n.Path: %v\n", nibbles, n.Path)
				return nil, fmt.Errorf("账户不存在（ExtensionNode 路径不匹配）")
			}
			// 如果匹配，继续查找子节点
			nibbles = nibbles[len(n.Path):]
			fmt.Printf("[Get] 匹配 ExtensionNode，剩余 nibbles：%v\n", nibbles)
			node = n.Child

		case *LeafNode:
			fmt.Printf("[Get] 当前节点类型：LeafNode，路径：%v，剩余 nibbles：%v，账户哈希：%x\n", n.Path, nibbles, n.Value)
			if bytes.Equal(n.Path, nibbles) {
				accountBytes, err := m.DB.Get(n.Value)
				if err != nil {
					return nil, fmt.Errorf("数据库中找不到账户数据，哈希：%x", n.Value)
				}
				return common.BytesToAccount(accountBytes), nil
			}
			return nil, fmt.Errorf("账户不存在（LeafNode 路径不匹配）")

		default:
			return nil, fmt.Errorf("未知节点类型")
		}
	}
}

func (m *MPT) Insert(key, value []byte) error {
	valueHash := SHA3_256(value)
	fmt.Printf("\n🔑 插入 Key: %x，值哈希: %x\n", key, valueHash)

	nibbles := BytesToNibbles(key)
	fmt.Printf("[Insert] 插入的 key 转为 nibbles: %v\n", nibbles)

	//存储值哈希映射
	if err := m.DB.Put(valueHash[:], value); err != nil {
		return err
	}

	// 空树情况
	if m.Root == nil {
		leaf := &LeafNode{
			Path:  nibbles,
			Value: valueHash[:],
		}
		m.StoreNode(leaf)
		m.Root = leaf
		fmt.Printf("🌱 Trie 为空，插入 LeafNode 作为根: Path=%x, Value=%x\n", leaf.Path, leaf.Value)
		return nil
	}

	// 查找最长前缀路径
	fmt.Printf("🔍 查找最长前缀路径...\n")
	path, nodes, err := m.FindLongestPrefix(nibbles)
	if err != nil {
		return fmt.Errorf("查找最长前缀路径失败: %v", err)
	}

	prefixNibbles := bytes.Join(path, nil)
	fmt.Printf("prefixNibbles: %x\n", prefixNibbles)
	fmt.Printf("nibbles: %x\n", nibbles)

	// 完全匹配情况
	if bytes.Equal(prefixNibbles, nibbles) {
		fmt.Printf("最后一个节点: %s\n", nodes[len(nodes)-1].GetType())
		leaf, ok := nodes[len(nodes)-1].(*LeafNode)
		if !ok {
			return errors.New("最后一个节点不是 LeafNode")
		}
		leaf.Value = valueHash[:]
		m.StoreNode(leaf)
		fmt.Printf("✅ 完全匹配，更新 LeafNode: Path=%x, NewValue=%x\n", leaf.Path, leaf.Value)
		return nil
	}

	// 部分匹配处理
	lastNode := nodes[len(nodes)-1]
	commonPrefixLen := m.prefixLength(nibbles, prefixNibbles)
	remainingKey := nibbles[commonPrefixLen:]
	remainingPath := prefixNibbles[commonPrefixLen:]

	fmt.Printf("🔀 部分匹配: 公共前缀长度=%d\n", commonPrefixLen)
	fmt.Printf("🆕 New LeafNode Path: %x\n", remainingKey)
	fmt.Printf("🧩 原节点剩余路径: %x\n", remainingPath)

	branch := &BranchNode{}
	var newNode Node = branch

	// 处理新键的剩余部分
	if len(remainingKey) > 0 {
		// 有剩余键，创建叶子节点
		index := remainingKey[0]
		if len(remainingKey) > 1 {
			// 剩余键长度>1，创建叶子节点
			newLeaf := &LeafNode{
				Path:  remainingKey[1:],
				Value: valueHash[:],
			}
			m.StoreNode(newLeaf)
			fmt.Printf("🔧 创建新 LeafNode Path=%x 加入 BranchNode Children[%x]\n", newLeaf.Path, index)
			branch.Children[index] = newLeaf
		} else {
			// 剩余键长度=1，在适当位置存储值
			branch.Children[index] = &LeafNode{
				Path:  []byte{},
				Value: valueHash[:],
			}
		}
	} else {
		// 无剩余键，直接设置值
		branch.Value = valueHash[:]
	}

	// 处理原节点
	switch last := lastNode.(type) {
	case *LeafNode:
		fmt.Printf("当前节点类型：LeafNode，路径：%x\n", last.Path)
		if commonPrefixLen == len(last.Path) {
			branch.Value = last.Value
		} else {
			// 直接使用已计算的 remainingPath
			fmt.Printf("🔄 LeafNode 剩余路径: %x\n", remainingPath)
			index := remainingPath[0]
			fmt.Printf("🔄 LeafNode 新分支索引: %x\n", index)
			newOldLeaf := &LeafNode{
				Path:  remainingPath[1:],
				Value: last.Value,
			}
			branch.Children[index] = newOldLeaf
			m.StoreNode(newOldLeaf)
			fmt.Printf("🔄 创建新 LeafNode Path=%x 加入 BranchNode Children[%x]\n", newOldLeaf.Path, index)
		}

		// 创建扩展节点
		if commonPrefixLen > 0 {
			// 计算当前层级需要压缩的路径部分
			existingPathLen := len(prefixNibbles) - len(last.Path)
			extPath := nibbles[existingPathLen:commonPrefixLen]
			ext := &ExtensionNode{
				Path:  extPath,
				Child: branch,
			}
			m.StoreNode(ext)
			newNode = ext
			fmt.Printf("🔧 新建 ExtensionNode: Path=%x (新压缩部分)\n", extPath)
		}

	case *ExtensionNode:
		fmt.Printf("当前节点类型：ExtensionNode，路径：%x\n", last.Path)
		realKeylen := len(prefixNibbles) - len(last.Path)// 已经匹配的长度
		fmt.Printf("🔍 已匹配长度: %d\n", realKeylen)
		leftNibbles := nibbles[realKeylen:] // 剩余的 nibbles
		fmt.Printf("🔍 剩余 nibbles: %x\n", leftNibbles)
		extCommonLen := m.prefixLength(leftNibbles, last.Path)

		if extCommonLen < len(last.Path) {
			// 原扩展节点需要拆分
			remainingOldPath := last.Path[extCommonLen:]
			index := remainingOldPath[0]

			// 1. 处理前面的公共前缀部分
			if extCommonLen > 0 {
				// 创建新的扩展节点表示公共前缀部分
				commonPath := last.Path[:extCommonLen]
				newExtension := &ExtensionNode{
					Path:  commonPath,
					Child: branch,
				}
				m.StoreNode(newExtension)
				newNode = newExtension // 更新当前节点为新的扩展节点
				fmt.Printf("🔄 保留扩展节点前半部分 Path=%x 作为新的 ExtensionNode\n", commonPath)
			}

			// 2. 处理后面的剩余路径部分
			if len(remainingOldPath) > 1 {
				// 分割原来的扩展节点的后部分创建新的 ExtensionNode，作为 BranchNode 的 child
				newExt := &ExtensionNode{
					Path:  remainingOldPath[1:], // 去掉第一个 nibble（用于分支）
					Child: last.Child,
				}
				m.StoreNode(newExt)
				branch.Children[index] = newExt
				fmt.Printf("🔄 创建新扩展节点 Path=%x 加入 BranchNode Children[%x]\n", newExt.Path, index)
			} else {
				// 剩余路径只有 1 个 nibble，直接接原扩展节点的 child
				branch.Children[index] = last.Child
				fmt.Printf("🔄 直接使用原扩展节点的子节点加入 BranchNode Children[%x]\n", index)
			}
		} else {
			// 原扩展节点路径被完全匹配，index 由新插入路径的下一个 nibble 决定
			if len(nibbles) > extCommonLen {
				index := nibbles[extCommonLen]
				branch.Children[index] = last.Child
				fmt.Printf("🔄 扩展节点路径完全匹配，使用其子节点加入 BranchNode Children[%x]\n", index)
			} else {
				// 插入路径完全匹配扩展路径，并且没有剩余 nibble
				branch.Value, err = last.Child.GetValue()
				if err != nil {
					return fmt.Errorf("获取原扩展节点子节点值失败: %v", err)
				}
				fmt.Printf("🔄 扩展节点路径和插入路径完全匹配，设置 Branch 的 Value=%x\n", branch.Value)
			}
		}

	case *BranchNode:
		fmt.Printf("当前节点类型：BranchNode\n")

		if len(remainingKey) == 0 {
			// 如果剩余键为空，直接更新分支节点的值
			branch.Value = last.Value // 保留原值
			for i := 0; i < 16; i++ {
				branch.Children[i] = last.Children[i] // 复制所有子节点
			}

			// 将新值设置到分支节点
			branch.Value = valueHash[:]
			fmt.Printf("🔄 更新 BranchNode 的值: %x\n", valueHash)
		} else {
			// 有剩余键，需要沿着分支节点继续处理
			// 复制原分支节点的所有信息
			branch.Value = last.Value
			for i := 0; i < 16; i++ {
				branch.Children[i] = last.Children[i]
			}

			// 处理剩余键
			index := remainingKey[0]
			if len(remainingKey) > 1 {
				// 如果剩余键长度大于1，创建新的叶子节点
				newLeaf := &LeafNode{
					Path:  remainingKey[1:],
					Value: valueHash[:],
				}
				m.StoreNode(newLeaf)
				branch.Children[index] = newLeaf
				fmt.Printf("🔧 在分支节点添加新叶子节点 Path=%x 于索引 %d\n", remainingKey[1:], index)
			} else {
				// 剩余键长度为1，创建空路径叶子节点
				emptyLeaf := &LeafNode{
					Path:  []byte{},
					Value: valueHash[:],
				}
				m.StoreNode(emptyLeaf)
				branch.Children[index] = emptyLeaf
				fmt.Printf("🔧 在分支节点添加空路径叶子节点于索引 %d\n", index)
			}
		}

		// 不需要创建新的扩展节点
		newNode = branch
	}

	m.StoreNode(branch)
	nodes[len(nodes)-1] = newNode

	// 回溯更新路径
	for i := len(nodes) - 2; i >= 0; i-- {
		parent := nodes[i]
		child := nodes[i+1]

		switch p := parent.(type) {
		case *ExtensionNode:
			p.Child = child
			m.StoreNode(p)
			nodes[i] = p
			fmt.Printf("🔁 回溯更新 ExtensionNode Path=%x 的子节点\n", p.Path)
		case *BranchNode:
			if i+1 < len(path) {
				index := path[i+1][0]
				p.Children[index] = child
				m.StoreNode(p)
				nodes[i] = p
				fmt.Printf("🔁 回溯更新 BranchNode Children[%x]\n", index)
			}
		}
	}

	// 更新根节点
	if len(nodes) > 0 {
		m.Root = nodes[0]
		fmt.Printf("🌳 更新 Root 节点类型: %T\n", m.Root)
	} else {
		m.Root = newNode
		fmt.Printf("🌳 设置 Root 为新节点类型: %T\n", m.Root)
	}

	return nil
}

// SetRoot 更新 MPT 的根哈希值，从数据库加载对应的根节点
func (m *MPT) SetRoot(root common.Hash) error {
    // 如果是空哈希，将根节点设为 nil
    if root == (common.Hash{}) {
        m.Root = nil
        return nil
    }

    // 从数据库加载根节点
    rootNode, err := m.LoadNode(root)
    if err != nil {
        return fmt.Errorf("加载根节点失败: %w", err)
    }

    // 更新 MPT 的根节点
    m.Root = rootNode
    return nil
}

// GetRootHash 返回当前 MPT 根节点的哈希值
func (m *MPT) GetRootHash() common.Hash {
    // 如果根节点为空，返回空哈希
    if m.Root == nil {
        return common.Hash{}
    }
    
    // 返回根节点的哈希
    return m.Root.GetHash()
}