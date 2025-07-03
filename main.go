package main

import (
	"chain/common"
	"chain/executor"
	"chain/kvstore/leveldb"
	"chain/statedb"
	"chain/transaction"
	"chain/trie/mpt"
	"chain/txpool"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
)

func main() {
	fmt.Println("🚀 启动区块链系统测试...")

	// 1. 设置测试环境
	dbPath := "./testdb_executor"
	os.RemoveAll(dbPath)
	defer os.RemoveAll(dbPath)

	kvStore, err := leveldb.NewLevelDB(dbPath)
	if err != nil {
		fmt.Printf("❌ 创建数据库失败: %v\n", err)
		return
	}
	defer kvStore.Close()

	mptrie, err := mpt.NewMPT(kvStore, common.Hash{})
	if err != nil {
		fmt.Printf("❌ 创建MPT失败: %v\n", err)
		return
	}

	stateDB := statedb.NewStateDB(mptrie)
	txPool := txpool.NewTxPool(stateDB)
	executor := executor.NewSimpleExecutor(stateDB, txPool)

	fmt.Println("✅ 系统组件初始化完成")

	// 2. 创建测试账户
	senderKey, _ := crypto.GenerateKey()
	senderPub := senderKey.Public().(*ecdsa.PublicKey)
	senderAddr := common.BytesToAddress(crypto.PubkeyToAddress(*senderPub).Bytes())
	receiverAddr := common.Address{1, 2, 3, 4, 5}

	fmt.Printf("📋 发送方地址: %x\n", senderAddr)
	fmt.Printf("📋 接收方地址: %x\n", receiverAddr)

	// 3. 初始化发送方账户 
	senderAccount := &common.Account{
		Address: senderAddr,
		Balance: 10000000000000000, 
		Nonce:   10,
	}
	err = stateDB.Put(senderAddr, senderAccount)
	if err != nil {
		fmt.Printf("❌ 初始化账户失败: %v\n", err)
		return
	}

	fmt.Printf("💰 发送方初始余额: %d wei\n", senderAccount.Balance)

	// 4. 创建交易
	tx := &transaction.Transaction{
		Txdata: transaction.Txdata{
			To:       receiverAddr,
			Nonce:    11,
			Value:    big.NewInt(1000),
			GasPrice: 1000, // 🌟 从20,000,000,000降低到1,000 wei
			GasLimit: 21000,
			Data:     []byte{},
			ChainID:  big.NewInt(1),
		},
		Signature: nil,
	}

	// 签名交易
	sig := tx.SetSignature(senderKey)
	if len(sig) == 0 {
		fmt.Println("❌ 交易签名失败")
		return
	}
	fmt.Println("✅ 交易签名成功")

	// 5. 添加到交易池
	err = txPool.AddTransaction(tx)
	if err != nil {
		fmt.Printf("❌ 添加交易到池失败: %v\n", err)
		return
	}
	fmt.Println("✅ 交易已添加到交易池")

	// 6. 执行交易
	fmt.Println("🔄 开始执行交易...")
	result := executor.ExecuteNextTransaction()

	if !result.Success {
		fmt.Printf("❌ 交易执行失败: %v\n", result.Error)
		return
	}

	fmt.Printf("✅ 交易执行成功! Gas使用: %d\n", result.GasUsed)

	// 验证Gas使用量
	if result.GasUsed != 21000 {
		fmt.Printf("⚠️ Gas使用量异常: 期望21000, 实际%d\n", result.GasUsed)
	} else {
		fmt.Println("✅ Gas使用量正确")
	}

	// 7. 验证状态变更
	finalSender, err := stateDB.Get(senderAddr)
	if err != nil {
		fmt.Printf("❌ 获取发送方账户失败: %v\n", err)
		return
	}

	finalReceiver, err := stateDB.Get(receiverAddr)
	if err != nil {
		fmt.Printf("❌ 获取接收方账户失败: %v\n", err)
		return
	}

	fmt.Printf("\n📊 最终状态:\n")
	fmt.Printf("发送方余额: %d wei\n", finalSender.Balance)
	fmt.Printf("接收方余额: %d wei\n", finalReceiver.Balance)

	// 使用uint64而不是big.Int
	expectedReceiverBalance := uint64(1000)
	if finalReceiver.Balance == expectedReceiverBalance {
		fmt.Println("✅ 接收方余额验证通过")
	} else {
		fmt.Printf("❌ 接收方余额错误: 期望 %d, 实际 %d\n", expectedReceiverBalance, finalReceiver.Balance)
	}

	// 计算预期的发送方余额 - 🌟 使用新的Gas价格
	gasCost := uint64(21000 * 1000) // 21,000,000 wei
	transferAmount := uint64(1000)
	expectedSenderBalance := uint64(10000000000000000) - transferAmount - gasCost

	if finalSender.Balance == expectedSenderBalance {
		fmt.Println("✅ 发送方余额验证通过")
	} else {
		fmt.Printf("❌ 发送方余额错误: 期望 %d, 实际 %d\n", expectedSenderBalance, finalSender.Balance)
	}

	fmt.Println("\n🎉 区块链系统测试完成!")
	fmt.Printf("转账金额: %d wei\n", transferAmount)
	fmt.Printf("Gas费用: %d wei\n", gasCost)
	fmt.Printf("总花费: %d wei\n", transferAmount+gasCost)
}
