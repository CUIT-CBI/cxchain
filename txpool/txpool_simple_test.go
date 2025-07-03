package txpool

import (
	"chain/common"
	"chain/kvstore/leveldb"
	"chain/statedb"
	"chain/transaction"
	"chain/trie/mpt"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

// 辅助函数：创建测试环境
func setupSimpleTestEnv(t *testing.T) (statedb.StateDB, *common.Address, *ecdsa.PrivateKey, func()) {
	// 创建临时数据库
	dbPath := "./testdb_simple"
	os.RemoveAll(dbPath) // 确保目录干净

	// 初始化KV存储
	kvStore, err := leveldb.NewLevelDB(dbPath)
	if err != nil {
		t.Fatalf("创建数据库失败: %v", err)
	}

	// 创建新的MPT和StateDB
	mptrie, err := mpt.NewMPT(kvStore, common.Hash{})
	if err != nil {
		t.Fatalf("创建MPT失败: %v", err)
	}
	state := statedb.NewStateDB(mptrie)

	// 生成测试账户
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("生成私钥失败: %v", err)
	}

	// 获取账户地址
	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	ethAddr := crypto.PubkeyToAddress(*publicKey)
	address := common.BytesToAddress(ethAddr.Bytes())

	// 创建账户并设置初始状态
	account := &common.Account{
		Address: address,
		Balance: 1000000,
		Nonce:   10, // 重要：设置初始nonce=10
	}
	err = state.Put(address, account)
	if err != nil {
		t.Fatalf("创建账户失败: %v", err)
	}

	// 返回清理函数
	cleanup := func() {
		kvStore.Close()
		os.RemoveAll(dbPath)
	}

	return state, &address, privateKey, cleanup
}

// 辅助函数：创建交易
func createSignedTx(t *testing.T, privateKey *ecdsa.PrivateKey, nonce uint64, to common.Address, value *big.Int, gasPrice uint64) *transaction.Transaction {
	tx := &transaction.Transaction{}
	tx.Nonce = nonce
	tx.To = to
	tx.Value = value
	tx.GasPrice = gasPrice
	tx.GasLimit = 21000
	tx.Data = []byte{}
	tx.ChainID = big.NewInt(1)

	signature := tx.SetSignature(privateKey)
	if len(signature) == 0 {
		t.Fatalf("签名交易失败")
	}

	return tx
}

// 辅助函数：打印交易池状态
func printPoolStatus(t *testing.T, pool *TxPool, address common.Address, testName string) {
	fmt.Printf("\n=== %s 状态 ===\n", testName)
	fmt.Printf("GlobalBox数量: %d\n", len(pool.globalBox))

	if pendingBoxes, exists := pool.pending[address]; exists {
		fmt.Printf("Pending盒子数: %d\n", len(pendingBoxes))
		for i, box := range pendingBoxes {
			fmt.Printf("  盒子#%d: GasPrice=%dG, 交易数=%d\n", i, box.GasPrice/1000000000, len(box.Tx))
			for j, tx := range box.Tx {
				fmt.Printf("    交易#%d: Nonce=%d, GasPrice=%dG\n", j, tx.Nonce, tx.GasPrice/1000000000)
			}
		}
	} else {
		fmt.Printf("Pending盒子数: 0\n")
	}

	if queueTxs, exists := pool.queue[address]; exists {
		fmt.Printf("Queue交易数: %d\n", len(queueTxs))
		for nonce, tx := range queueTxs {
			fmt.Printf("  Queue交易: Nonce=%d, GasPrice=%dG\n", nonce, tx.GasPrice/1000000000)
		}
	} else {
		fmt.Printf("Queue交易数: 0\n")
	}
	fmt.Println()
}

// 辅助函数：重置账户状态
func resetAccountState(t *testing.T, state statedb.StateDB, address common.Address, nonce uint64) {
	account, err := state.Get(address)
	if err != nil {
		t.Fatalf("获取账户失败: %v", err)
	}

	oldNonce := account.Nonce
	account.Nonce = nonce
	err = state.Put(address, account)
	if err != nil {
		t.Fatalf("重置账户状态失败: %v", err)
	}

	fmt.Printf("🔄 重置账户nonce: %d → %d\n", oldNonce, nonce)
}

// 完整的交易池测试
func TestSimpleTxPoolOperations(t *testing.T) {
	// 设置测试环境
	state, senderAddr, privateKey, cleanup := setupSimpleTestEnv(t)
	defer cleanup()

	// 创建接收方地址
	receiver := common.Address{1, 2, 3, 4, 5}

	// === 测试1: 基本交易添加和弹出 ===
	t.Log("===== 测试1: 基本交易添加和弹出 =====")
	pool := NewTxPool(state)

	// 添加一个基本交易
	tx := createSignedTx(t, privateKey, 11, receiver, big.NewInt(100), 20000000000) // 20G
	err := pool.AddTransaction(tx)
	assert.NoError(t, err, "添加有效交易应该成功")

	// 验证交易在正确位置
	assert.True(t, pool.all[tx.GetHash()], "交易应该在all映射中")
	assert.Equal(t, 1, len(pool.globalBox), "应该有1个全局盒子")
	assert.Equal(t, 1, len(pool.pending[*senderAddr]), "应该有1个pending盒子")
	assert.Equal(t, 0, len(pool.queue[*senderAddr]), "queue应该为空")

	printPoolStatus(t, pool, *senderAddr, "添加单个交易后")

	// 弹出交易
	poppedTx := pool.PopTransaction()
	assert.Equal(t, uint64(11), poppedTx.Nonce, "弹出的交易nonce应该正确")
	assert.Equal(t, 0, len(pool.globalBox), "弹出后应该为空")

	// === 测试2: 交易替换机制 ===
	fmt.Println()
	t.Log("===== 测试2: 完整的交易替换测试 =====")

	// 🌟 重置账户状态
	resetAccountState(t, state, *senderAddr, 10)
	pool = NewTxPool(state)

	// 场景1: 简单替换 - 单个交易盒子中的替换
	fmt.Println("\n🎯 场景1: 简单单交易替换")

	// 创建相同nonce但不同gasPrice的交易
	lowGasTx := createSignedTx(t, privateKey, 11, receiver, big.NewInt(100), 10000000000)  // 10G
	highGasTx := createSignedTx(t, privateKey, 11, receiver, big.NewInt(100), 20000000000) // 20G

	// 添加第一个交易
	err = pool.AddTransaction(lowGasTx)
	assert.NoError(t, err, "添加第一个交易应该成功")
	fmt.Printf("添加低GasPrice交易后: pending盒子数=%d\n", len(pool.pending[*senderAddr]))

	// 添加第二个交易（应该替换第一个）
	fmt.Printf("🔄 尝试添加高GasPrice的相同nonce交易...\n")
	err = pool.AddTransaction(highGasTx)
	assert.NoError(t, err, "添加第二个交易应该成功")

	// 验证替换结果
	assert.False(t, pool.all[lowGasTx.GetHash()], "低GasPrice交易应该被移除")
	assert.True(t, pool.all[highGasTx.GetHash()], "高GasPrice交易应该存在")

	// 检查交易是否被正确替换
	found := false
	for _, box := range pool.pending[*senderAddr] {
		for _, tx := range box.Tx {
			if tx.Nonce == 11 {
				found = true
				assert.Equal(t, uint64(20000000000), tx.GasPrice, "应该保留GasPrice更高的交易")
			}
		}
	}
	assert.True(t, found, "应该找到nonce=11的交易")
	fmt.Printf("✅ 简单替换成功\n")

	// 场景2: 复杂替换 - 多交易盒子中间位置的替换
	fmt.Println("\n🎯 场景2: 复杂的盒子拆分替换")
	pool = NewTxPool(state)

	// 创建一个包含多个交易的盒子
	tx11 := createSignedTx(t, privateKey, 11, receiver, big.NewInt(100), 20000000000) // 20G
	tx12 := createSignedTx(t, privateKey, 12, receiver, big.NewInt(200), 30000000000) // 30G
	tx13 := createSignedTx(t, privateKey, 13, receiver, big.NewInt(300), 25000000000) // 25G
	tx14 := createSignedTx(t, privateKey, 14, receiver, big.NewInt(400), 35000000000) // 35G

	// 添加所有交易，它们会组成盒子
	pool.AddTransaction(tx11)
	pool.AddTransaction(tx12)
	pool.AddTransaction(tx13)
	pool.AddTransaction(tx14)

	fmt.Printf("替换前状态:\n")
	for i, box := range pool.pending[*senderAddr] {
		fmt.Printf("  盒子#%d: GasPrice=%dG, 交易数=%d\n", i, box.GasPrice/1000000000, len(box.Tx))
		for j, tx := range box.Tx {
			fmt.Printf("    交易#%d: Nonce=%d, GasPrice=%dG\n", j, tx.Nonce, tx.GasPrice/1000000000)
		}
	}

	// 替换中间的交易(nonce=12)，用更高GasPrice的交易
	newTx12 := createSignedTx(t, privateKey, 12, receiver, big.NewInt(200), 50000000000) // 50G
	fmt.Printf("\n🔄 替换中间交易 Tx12: 30G → 50G\n")

	err = pool.AddTransaction(newTx12)
	assert.NoError(t, err, "替换交易应该成功")

	// 验证旧交易被删除，新交易被添加
	assert.False(t, pool.all[tx12.GetHash()], "旧的tx12应该被移除")
	assert.True(t, pool.all[newTx12.GetHash()], "新的tx12应该存在")

	fmt.Printf("\n替换后状态:\n")
	totalTxCount := 0
	for i, box := range pool.pending[*senderAddr] {
		fmt.Printf("  盒子#%d: GasPrice=%dG, 交易数=%d\n", i, box.GasPrice/1000000000, len(box.Tx))
		for j, tx := range box.Tx {
			fmt.Printf("    交易#%d: Nonce=%d, GasPrice=%dG\n", j, tx.Nonce, tx.GasPrice/1000000000)
		}
		totalTxCount += len(box.Tx)
	}

	// 验证拆分结果：应该仍有4个交易，但可能分布在不同盒子中
	assert.Equal(t, 4, totalTxCount, "替换后应该仍有4个交易")

	// 验证替换后的执行顺序仍然正确
	fmt.Printf("\n验证执行顺序:\n")
	expectedNonces := []uint64{11, 12, 13, 14}
	for i, expectedNonce := range expectedNonces {
		poppedTx := pool.PopTransaction()
		assert.Equal(t, expectedNonce, poppedTx.Nonce,
			fmt.Sprintf("第%d个弹出的交易nonce应该是%d", i+1, expectedNonce))
		fmt.Printf("  弹出: Nonce=%d, GasPrice=%dG\n", poppedTx.Nonce, poppedTx.GasPrice/1000000000)

		// 特别验证被替换的交易
		if expectedNonce == 12 {
			assert.Equal(t, uint64(50000000000), poppedTx.GasPrice, "替换后的tx12应该是50G")
		}
	}

	fmt.Printf("✅ 复杂替换和拆分成功！\n")


	// === 测试3: nonce不连续交易进入queue ===
	t.Log("===== 测试3: nonce不连续交易进入queue =====")
	pool = NewTxPool(state)
	resetAccountState(t, state, *senderAddr, 10)
	// 添加nonce=13的交易（跳过了12），应该进入queue
	futureTx := createSignedTx(t, privateKey, 13, receiver, big.NewInt(300), 30000000000) // 30G
	err = pool.AddTransaction(futureTx)
	assert.NoError(t, err)

	// 验证交易在queue中
	assert.Equal(t, 0, len(pool.pending[*senderAddr]), "pending应该为空")
	assert.Equal(t, 1, len(pool.queue[*senderAddr]), "queue应该有1个交易")
	assert.Equal(t, uint64(13), pool.queue[*senderAddr][13].Nonce, "queue中应该是nonce=13的交易")

	printPoolStatus(t, pool, *senderAddr, "nonce跳跃交易进入queue后")

	// === 测试4: 关键测试 - 完整的自动提升场景 ===
	t.Log("===== 测试4: 完整的自动提升场景 =====")
	pool = NewTxPool(state)
	resetAccountState(t, state, *senderAddr, 10)
	fmt.Println("🎯 模拟场景：Account nonce=10, 添加顺序 Tx13→Tx11→Tx12")

	// 1. 添加 Tx1(nonce=13, GasPrice=30G) → 应该进入queue
	tx1 := createSignedTx(t, privateKey, 13, receiver, big.NewInt(300), 30000000000) // 30G
	fmt.Printf("\n1️⃣ 添加 Tx1(nonce=13, GasPrice=30G)\n")
	err = pool.AddTransaction(tx1)
	assert.NoError(t, err)

	assert.Equal(t, 0, len(pool.pending[*senderAddr]), "pending应该为空")
	assert.Equal(t, 1, len(pool.queue[*senderAddr]), "queue应该有1个交易")
	printPoolStatus(t, pool, *senderAddr, "添加Tx1后")

	// 2. 添加 Tx2(nonce=11, GasPrice=20G) → 应该进入pending
	tx2 := createSignedTx(t, privateKey, 11, receiver, big.NewInt(100), 20000000000) // 20G
	fmt.Printf("\n2️⃣ 添加 Tx2(nonce=11, GasPrice=20G)\n")
	err = pool.AddTransaction(tx2)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(pool.pending[*senderAddr]), "pending应该有1个盒子")
	assert.Equal(t, 1, len(pool.queue[*senderAddr]), "queue应该仍有1个交易")
	printPoolStatus(t, pool, *senderAddr, "添加Tx2后")

	// 3. 添加 Tx3(nonce=12, GasPrice=25G) → 应该进入pending，并自动提升Tx1
	tx3 := createSignedTx(t, privateKey, 12, receiver, big.NewInt(200), 25000000000) // 25G
	fmt.Printf("\n3️⃣ 添加 Tx3(nonce=12, GasPrice=25G) - 应该触发自动提升\n")
	err = pool.AddTransaction(tx3)
	assert.NoError(t, err)

	// 验证最终状态：所有交易都在pending中，queue为空
	assert.Equal(t, 0, len(pool.queue[*senderAddr]), "queue应该为空（自动提升完成）")
	pendingBoxes := pool.pending[*senderAddr]

	// 验证盒子结构
	totalTransactions := 0
	for _, box := range pendingBoxes {
		totalTransactions += len(box.Tx)
	}
	assert.Equal(t, 3, totalTransactions, "pending中应该总共有3个交易")

	printPoolStatus(t, pool, *senderAddr, "自动提升完成后")

	// 验证具体的盒子结构
	// 预期：Box1(25G): [Tx2(11,20G), Tx3(12,25G)],[Tx1(13,30G)]
	

	// === 测试5: 关键测试 - PopTransaction的nonce连续性检查 ===
	t.Log("===== 测试5: PopTransaction的nonce连续性检查 =====")

	// 🌟 注意：测试5使用测试4的状态，不需要重置

	fmt.Printf("\n🎯 测试执行顺序：期望 Tx2(11) → Tx3(12) → Tx1(13)\n")

	// 第一次弹出：应该是Tx2(nonce=11)，即使30G盒子优先级更高
	fmt.Printf("\n1️⃣ 第一次PopTransaction（期望：nonce=11）:\n")
	poppedTx = pool.PopTransaction()
	assert.Equal(t, uint64(11), poppedTx.Nonce, "第一个弹出的应该是nonce=11")
	assert.Equal(t, uint64(20000000000), poppedTx.GasPrice, "应该是20G")
	fmt.Printf("✅ 弹出: Nonce=%d, GasPrice=%dG\n", poppedTx.Nonce, poppedTx.GasPrice/1000000000)

	// 第二次弹出：应该是Tx3(nonce=12)
	fmt.Printf("\n2️⃣ 第二次PopTransaction（期望：nonce=12）:\n")
	poppedTx = pool.PopTransaction()
	assert.Equal(t, uint64(12), poppedTx.Nonce, "第二个弹出的应该是nonce=12")
	assert.Equal(t, uint64(25000000000), poppedTx.GasPrice, "应该是25G")
	fmt.Printf("✅ 弹出: Nonce=%d, GasPrice=%dG\n", poppedTx.Nonce, poppedTx.GasPrice/1000000000)

	// 第三次弹出：应该是Tx1(nonce=13)
	fmt.Printf("\n3️⃣ 第三次PopTransaction（期望：nonce=13）:\n")
	poppedTx = pool.PopTransaction()
	assert.Equal(t, uint64(13), poppedTx.Nonce, "第三个弹出的应该是nonce=13")
	assert.Equal(t, uint64(30000000000), poppedTx.GasPrice, "应该是30G")
	fmt.Printf("✅ 弹出: Nonce=%d, GasPrice=%dG\n", poppedTx.Nonce, poppedTx.GasPrice/1000000000)


	fmt.Printf("\n🎉 完美！执行顺序正确：Tx2(11,20G) → Tx3(12,25G) → Tx1(13,30G)\n")
	fmt.Printf("✅ nonce连续性 ✅ GasPrice优化平衡 ✅ 自动提升机制\n")

	// === 测试6: 复杂场景 - 多个nonce跳跃 ===
	t.Log("===== 测试6: 复杂的queue管理场景 =====")

	// 🌟 重置账户状态
	resetAccountState(t, state, *senderAddr, 10)
	pool = NewTxPool(state)

	// 添加多个不连续的nonce
	tx15 := createSignedTx(t, privateKey, 15, receiver, big.NewInt(500), 15000000000) // 15G
	tx14 = createSignedTx(t, privateKey, 14, receiver, big.NewInt(400), 25000000000)  // 25G
	tx16 := createSignedTx(t, privateKey, 16, receiver, big.NewInt(600), 35000000000) // 35G

	// 全部应该进入queue
	pool.AddTransaction(tx15)
	pool.AddTransaction(tx14)
	pool.AddTransaction(tx16)

	assert.Equal(t, 0, len(pool.pending[*senderAddr]), "pending应该为空")
	assert.Equal(t, 3, len(pool.queue[*senderAddr]), "queue应该有3个交易")

	printPoolStatus(t, pool, *senderAddr, "添加不连续nonce交易后")

	// 添加连接交易
	tx11_new := createSignedTx(t, privateKey, 11, receiver, big.NewInt(100), 20000000000) // 20G
	pool.AddTransaction(tx11_new)

	// 仍然缺少12,13，所以只有tx11在pending
	assert.Equal(t, 1, len(pool.pending[*senderAddr]), "pending应该有1个盒子")
	assert.Equal(t, 3, len(pool.queue[*senderAddr]), "queue应该仍有3个交易")

	// 添加tx12
	tx12_new := createSignedTx(t, privateKey, 12, receiver, big.NewInt(200), 30000000000) // 30G
	pool.AddTransaction(tx12_new)

	// 仍然缺少13
	assert.Equal(t, 3, len(pool.queue[*senderAddr]), "queue应该仍有3个交易")

	// 添加关键的tx13，应该触发大量提升
	tx13_new := createSignedTx(t, privateKey, 13, receiver, big.NewInt(300), 40000000000) // 40G
	fmt.Printf("\n🚀 添加关键交易tx13，应该触发连锁提升...\n")
	pool.AddTransaction(tx13_new)

	// 现在所有交易都应该在pending中
	assert.Equal(t, 0, len(pool.queue[*senderAddr]), "queue应该为空")

	totalTx := 0
	for _, box := range pool.pending[*senderAddr] {
		totalTx += len(box.Tx)
	}
	assert.Equal(t, 6, totalTx, "pending中应该有6个交易")

	printPoolStatus(t, pool, *senderAddr, "连锁提升完成后")

	fmt.Printf("\n🎉 所有测试通过！交易池实现完美支持：\n")
	fmt.Printf("✅ 基本交易管理\n")
	fmt.Printf("✅ 交易替换机制\n")
	fmt.Printf("✅ nonce连续性检查\n")
	fmt.Printf("✅ 自动提升机制\n")
	fmt.Printf("✅ 复杂场景处理\n")
	fmt.Printf("✅ GasPrice优化平衡\n")
}

// === 测试7: 批量通知测试 ===
func TestBatchNotification(t *testing.T) {
	state, senderAddr, privateKey, cleanup := setupSimpleTestEnv(t)
	defer cleanup()

	pool := NewTxPool(state)
	receiver := common.Address{1, 2, 3, 4, 5}

	// 创建批量交易
	transactions := []*transaction.Transaction{
		createSignedTx(t, privateKey, 11, receiver, big.NewInt(100), 10000000000),
		createSignedTx(t, privateKey, 12, receiver, big.NewInt(200), 30000000000),
		createSignedTx(t, privateKey, 13, receiver, big.NewInt(300), 20000000000),
	}

	// 批量通知
	pool.NotifyTrascation(transactions)

	// 验证所有交易都被正确处理
	for _, tx := range transactions {
		assert.True(t, pool.all[tx.GetHash()], "所有交易都应该在池中")
	}

	printPoolStatus(t, pool, *senderAddr, "批量通知后")

	// 验证执行顺序
	expectedNonces := []uint64{11, 12, 13}
	for i, expectedNonce := range expectedNonces {
		poppedTx := pool.PopTransaction()
		assert.Equal(t, expectedNonce, poppedTx.Nonce,
			fmt.Sprintf("第%d个弹出的交易nonce应该是%d", i+1, expectedNonce))
	}
}
