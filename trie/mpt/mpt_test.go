package mpt_test

import (
	"os"
	"strings"
	"testing"
	"chain/trie/mpt"
	"chain/common"
	"chain/kvstore/leveldb"
)

func TestMPTWithAccounts(t *testing.T) {
	// 测试目录名
	testDir := "mpt_test_db"

	// 删除旧数据库
	os.RemoveAll(testDir)

	// 初始化 LevelDB
	db, err := leveldb.NewLevelDB(testDir)
	if err != nil {
		t.Fatalf("无法创建 LevelDB 实例: %v", err)
	}
	// 确保测试结束后清理
	defer func() {
		db.Close()
		os.RemoveAll(testDir)
	}()

	// 空 root hash 创建 MPT
	emptyHash := [32]byte{}
	tree, err := mpt.NewMPT(db, emptyHash)
	if err != nil {
		t.Fatalf("无法创建 MPT 实例: %v", err)
	}

	t.Log("==================================================")
	t.Log("=============== 测试模拟账户地址 =================")
	t.Log("==================================================")

	// 模拟账户组 - 使用描述性变量名
	addrAlice := common.BytesToAddress([]byte("addr00000000000000001"))
	addrBob := common.BytesToAddress([]byte("addr00000000000000002"))
	addrCharlie := common.BytesToAddress([]byte("addr00000000000000113"))
	addrDave := common.BytesToAddress([]byte("addr10000000000000007"))

	t.Logf("Alice地址: %x", addrAlice)
	t.Logf("Bob地址:   %x", addrBob)
	t.Logf("Charlie地址: %x", addrCharlie)
	t.Logf("Dave地址:    %x", addrDave)

	// 测试插入账户
	t.Run("InsertAccounts", func(t *testing.T) {
		// 插入Alice账户
		t.Log("\n【插入】Alice账户 - 余额1000")
		insertAccount(t, tree, addrAlice, &common.Account{
			Address: addrAlice,
			Balance: 1000,
			Nonce:   1,
		})

		t.Log("\n【MPT结构】插入Alice后")
		logMPT(t, tree)

		// 插入Charlie账户
		t.Log("\n【插入】Charlie账户 - 余额1500")
		insertAccount(t, tree, addrCharlie, &common.Account{
			Address: addrCharlie,
			Balance: 1500,
			Nonce:   1,
		})

		t.Log("\n【MPT结构】插入Charlie后")
		logMPT(t, tree)

		// 插入Bob账户
		t.Log("\n【插入】Bob账户 - 余额2000")
		insertAccount(t, tree, addrBob, &common.Account{
			Address: addrBob,
			Balance: 2000,
			Nonce:   2,
		})

		t.Log("\n【MPT结构】插入Bob后")
		logMPT(t, tree)

		// 插入Dave账户
		t.Log("\n【插入】Dave账户 - 余额2500")
		insertAccount(t, tree, addrDave, &common.Account{
			Address: addrDave,
			Balance: 2500,
			Nonce:   3,
		})

		t.Log("\n【MPT结构】插入Dave后")
		logMPT(t, tree)

		// 获取并打印根哈希
		rootHash := tree.GetRootHash()
		t.Logf("\n插入完成，MPT 根哈希: %x", rootHash)
	})

	// 测试读取账户
	t.Run("GetAccounts", func(t *testing.T) {
		t.Log("\n==================================================")
		t.Log("============== 测试读取模拟账户 ==================")
		t.Log("==================================================")

		testGetAccount(t, tree, addrAlice, "Alice")
		testGetAccount(t, tree, addrBob, "Bob")
		testGetAccount(t, tree, addrCharlie, "Charlie")
		testGetAccount(t, tree, addrDave, "Dave")

		// 验证账户数据正确性
		verifyAccount(t, tree, addrAlice, 1000, 1)
		verifyAccount(t, tree, addrBob, 2000, 2)
		verifyAccount(t, tree, addrCharlie, 1500, 1)
		verifyAccount(t, tree, addrDave, 2500, 3)
	})

	// 测试更新账户
	t.Run("UpdateAccounts", func(t *testing.T) {
		t.Log("\n==================================================")
		t.Log("================ 测试更新账户 ====================")
		t.Log("==================================================")
		t.Log("【更新】Bob账户 - 余额从2000增加到3000")

		insertAccount(t, tree, addrBob, &common.Account{
			Address: addrBob,
			Balance: 3000,
			Nonce:   2,
		})

		t.Log("\n【读取】更新后的Bob账户")
		testGetAccount(t, tree, addrBob, "Bob")

		// 验证更新后的账户数据
		verifyAccount(t, tree, addrBob, 3000, 2)

		t.Log("\n【MPT结构】更新Bob后")
		logMPT(t, tree)
	})

	// 测试不存在的账户
	t.Run("NonExistentAccount", func(t *testing.T) {
		t.Log("\n==================================================")
		t.Log("============= 测试不存在的账户 ===================")
		t.Log("==================================================")

		fakeAddr := common.BytesToAddress([]byte("addr00000000000000999"))
		t.Logf("不存在的账户地址: %x", fakeAddr)

		account, err := tree.Get(fakeAddr[:])
		if err != nil {
			t.Logf("预期错误: %v", err)
		} else if account != nil {
			t.Errorf("预期账户不存在，但找到了账户: %+v", account)
		} else {
			t.Log("测试通过: 不存在的账户返回nil")
		}
	})
}

// insertAccount 函数插入账户到 MPT
func insertAccount(t *testing.T, tree *mpt.MPT, addr common.Address, acc *common.Account) {
	key := addr[:]
	value := acc.Bytes()

	if err := tree.Insert(key, value); err != nil {
		t.Fatalf("插入账户 %x 失败: %v", addr, err)
	}
	t.Logf("成功插入账户: %x", addr)
}

// testGetAccount 函数测试读取账户
func testGetAccount(t *testing.T, tree *mpt.MPT, addr common.Address, accountName string) {
	account, err := tree.Get(addr[:])
	if err != nil {
		t.Errorf("【读取失败】%s账户 (%x): %v", accountName, addr, err)
		return
	}

	t.Logf("【读取成功】%s账户 (%x)", accountName, addr)
	t.Logf("  余额: %d", account.Balance)
	t.Logf("  Nonce: %d", account.Nonce)
}

// verifyAccount 函数验证账户数据是否符合预期
func verifyAccount(t *testing.T, tree *mpt.MPT, addr common.Address, expectedBalance uint64, expectedNonce uint64) {
	account, err := tree.Get(addr[:])
	if err != nil {
		t.Errorf("验证账户 %x 失败: %v", addr, err)
		return
	}

	if account.Balance != expectedBalance {
		t.Errorf("账户 %x 余额不符合预期: 得到 %d, 期望 %d",
			addr, account.Balance, expectedBalance)
	}

	if account.Nonce != expectedNonce {
		t.Errorf("账户 %x Nonce不符合预期: 得到 %d, 期望 %d",
			addr, account.Nonce, expectedNonce)
	}
}

// logMPT 函数记录 MPT 结构
func logMPT(t *testing.T, tree *mpt.MPT) {
	t.Log("MPT 结构打印:")
	logNode(t, tree.Root, 0)
}

// logNode 函数递归记录 MPT 节点
func logNode(t *testing.T, node mpt.Node, depth int) {
	if node == nil {
		t.Logf("%s<nil>", strings.Repeat("  ", depth))
		return
	}

	switch n := node.(type) {
	case *mpt.ExtensionNode:
		t.Logf("%s扩展节点 (Path=%x)", strings.Repeat("  ", depth), n.Path)
		logNode(t, n.Child, depth+1)

	case *mpt.BranchNode:
		t.Logf("%s分支节点", strings.Repeat("  ", depth))
		if len(n.Value) > 0 {
			t.Logf("%s  值: %x", strings.Repeat("  ", depth), n.Value)
		}

		for i, child := range n.Children {
			if child != nil {
				t.Logf("%s  [%x]:", strings.Repeat("  ", depth), i)
				logNode(t, child, depth+2)
			}
		}

	case *mpt.LeafNode:
		t.Logf("%s叶子节点 (Path=%x, Value=%x)",
			strings.Repeat("  ", depth), n.Path, n.Value)

	default:
		t.Logf("%s未知节点类型: %T", strings.Repeat("  ", depth), node)
	}
}
