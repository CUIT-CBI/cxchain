package executor

import (
	"chain/common"
	"chain/statedb"
	"chain/transaction"
	"chain/txpool"
	"fmt"
)

// SimpleExecutor 简单执行器
type SimpleExecutor struct {
	stateDB statedb.StateDB
	txPool  *txpool.TxPool
}

// ExecutionResult 简单执行结果
type ExecutionResult struct {
	TxHash  common.Hash
	Success bool
	GasUsed uint64
	Error   error
}

// NewSimpleExecutor 创建简单执行器
func NewSimpleExecutor(stateDB statedb.StateDB, txPool *txpool.TxPool) *SimpleExecutor {
	return &SimpleExecutor{
		stateDB: stateDB,
		txPool:  txPool,
	}
}

// ExecuteNextTransaction 执行下一个交易
func (e *SimpleExecutor) ExecuteNextTransaction() *ExecutionResult {
	// 1. 从交易池获取下一个交易
	tx := e.txPool.PopTransaction()
	if tx.Nonce == 0 {
		return &ExecutionResult{
			Success: false,
			Error:   fmt.Errorf("no more transactions in pool"),
		}
	}

	fmt.Printf("🔄 执行交易: Nonce=%d, From=%x, To=%x, Value=%s\n",
		tx.Nonce, tx.From(), tx.To, tx.Value.String())

	// 2. 执行交易
	return e.executeTransaction(&tx)
}

// executeTransaction 执行单个交易
func (e *SimpleExecutor) executeTransaction(tx *transaction.Transaction) *ExecutionResult {
	result := &ExecutionResult{
		TxHash:  tx.GetHash(),
		Success: false,
		GasUsed: 21000, // 基础gas
	}

	// 1. 获取发送方账户
	from := tx.From()
	fromAccount, err := e.stateDB.Get(from)
	if err != nil {
		result.Error = fmt.Errorf("获取发送方账户失败: %v", err)
		return result
	}

	// 2. 验证余额 - 🌟 简化为uint64操作
	value := tx.Value.Uint64()
	gasCost := tx.GasLimit * tx.GasPrice
	totalCost := value + gasCost

	if fromAccount.Balance < totalCost {
		result.Error = fmt.Errorf("余额不足: 需要%d, 拥有%d", totalCost, fromAccount.Balance)
		return result
	}

	// 3. 获取接收方账户
	toAccount, err := e.stateDB.Get(tx.To)
	if err != nil {
		// 创建新账户
		toAccount = &common.Account{
			Address: tx.To,
			Balance: 0,
			Nonce:   0,
		}
	}

	// 4. 执行转账 - 🌟 简化的uint64操作
	fromAccount.Balance -= totalCost
	toAccount.Balance += value

	// 5. 更新状态
	err = e.stateDB.Put(from, fromAccount)
	if err != nil {
		result.Error = fmt.Errorf("更新发送方账户失败: %v", err)
		return result
	}

	err = e.stateDB.Put(tx.To, toAccount)
	if err != nil {
		result.Error = fmt.Errorf("更新接收方账户失败: %v", err)
		return result
	}

	result.Success = true
	fmt.Printf("✅ 交易执行成功: 转账%d, Gas费用%d\n", value, gasCost)

	return result
}

// ExecuteBlock 执行一个区块的所有交易
func (e *SimpleExecutor) ExecuteBlock(maxTxCount int) ([]*ExecutionResult, error) {
	var results []*ExecutionResult

	fmt.Printf("🎯 开始执行区块，最大交易数: %d\n", maxTxCount)

	for i := 0; i < maxTxCount; i++ {
		result := e.ExecuteNextTransaction()

		if result.Error != nil && result.Error.Error() == "no more transactions in pool" {
			fmt.Printf("📭 交易池已空，执行了 %d 个交易\n", i)
			break
		}

		results = append(results, result)

		if !result.Success {
			fmt.Printf("❌ 交易执行失败: %v\n", result.Error)
			// 可以选择继续或停止
		}
	}

	fmt.Printf("🎉 区块执行完成，总计 %d 个交易\n", len(results))
	return results, nil
}

// GetAccountBalance 查询账户余额 - 🌟 返回uint64
func (e *SimpleExecutor) GetAccountBalance(addr common.Address) (uint64, error) {
	account, err := e.stateDB.Get(addr)
	if err != nil {
		return 0, err
	}
	return account.Balance, nil
}

// GetAccountNonce 查询账户nonce
func (e *SimpleExecutor) GetAccountNonce(addr common.Address) (uint64, error) {
	account, err := e.stateDB.Get(addr)
	if err != nil {
		return 0, err
	}
	return account.Nonce, nil
}
