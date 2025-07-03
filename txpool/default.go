package txpool

import (
	"chain/common"
	"chain/statedb"
	"chain/transaction"
	"errors"
	"fmt"
	"sort"
)

type TxPool struct {
	statedb   statedb.StateDB
	all       map[common.Hash]bool
	pending   map[common.Address]transaction.Boxes
	queue     map[common.Address]map[uint64]transaction.Transaction
	globalBox transaction.Boxes //交易执行的顺序
}

func NewTxPool(statedb statedb.StateDB) *TxPool {
	return &TxPool{
		statedb:   statedb,
		all:       make(map[common.Hash]bool),                                  // 存储所有交易的哈希
		pending:   make(map[common.Address]transaction.Boxes),                  // 存储每个地址的待处理交易盒子
		queue:     make(map[common.Address]map[uint64]transaction.Transaction), // 存储每个地址的队列交易
		globalBox: transaction.Boxes{},                                         // 全局的交易盒子
	}
}

func (p *TxPool) NotifyTrascation(txs []*transaction.Transaction) {
	// 遍历所有的交易
	for _, tx := range txs {
		// 添加交易到交易池
		if err := p.AddTransaction(tx); err != nil {
			// 如果添加交易失败，打印错误信息
			fmt.Printf("Failed to add transaction %s: %v\n", tx.GetHash(), err)
			continue
		}
	}
	return
}

func (p *TxPool) AddTransaction(tx *transaction.Transaction) error {
	// Check if the transaction is already in the pool
	if _, exists := p.all[tx.GetHash()]; exists {
		return errors.New("transaction already exists in the pool")
	}

	//get the transaction address
	from := tx.From()
	if from == (common.Address{}) {
		return errors.New("invalid transaction: missing sender address")
	}
	//get the account state from the statedb
	account, err := p.statedb.Get(from)
	if err != nil {
		return errors.New("failed to get account state: " + err.Error())
	}
	if account.Nonce >= tx.Nonce {
		return errors.New("transaction nonce is too low")
	}

	//get the pending transactions for the sender address
	pendingBoxes := p.pending[from]
	nonce := account.Nonce

	if len(pendingBoxes) > 0 {
		//和pending的最后一个交易进行比较
		//调试信息
		fmt.Print("🌟Pending boxes len: ", len(pendingBoxes))
		fmt.Println()
		lastBox := pendingBoxes[len(pendingBoxes)-1]
		lastTx := lastBox.Tx[len(lastBox.Tx)-1]
		nonce = lastTx.Nonce
	}
	// 如果交易的nonce大于pending的最后一个交易的nonce放入queue中
	if tx.Nonce > nonce+1 {
		//调试信息
		fmt.Printf("🚀 Adding transaction to queue")
		p.addQueueTransaction(tx)
	} else if tx.Nonce == nonce+1 {
		// 如果交易的nonce等于pending的最后一个交易的nonce+1，放入pending中
		// 调试信息
		fmt.Printf("✅ Adding transaction to pending")
		p.addPendingTransaction(pendingBoxes, tx)
	} else {
		// 如果交易的nonce小于等于pending的最后一个交易的nonce，替换pending中的交易
		// 调试信息
		fmt.Printf("🔄 Attempting to replace transaction: Nonce=%d\n", tx.Nonce)
		err := p.replace(pendingBoxes, tx)
		if err != nil {
			fmt.Printf("❌ Replace failed: %v\n", err)
			return err
		}
	}
	// Mark the transaction as existing in the pool
	p.all[tx.GetHash()] = true

	return nil
}

func (p *TxPool) PopTransaction() transaction.Transaction {
	if len(p.globalBox) == 0 {
		fmt.Println("没有交易盒子可供弹出")
		return transaction.Transaction{}
	}

	// 🌟 核心修复：遍历寻找可执行的交易（nonce连续的）
	for i, box := range p.globalBox {
		if len(box.Tx) == 0 {
			continue
		}

		firstTx := box.Tx[0]
		from := firstTx.From()

		// 检查这个交易是否可以执行（nonce连续性）
		if p.canExecuteTransaction(from, firstTx.Nonce) {
			fmt.Printf("🔍 弹出可执行交易: Nonce=%d, GasPrice=%d\n", firstTx.Nonce, firstTx.GasPrice)

			// 从盒子中删除该交易
			box.Tx = box.Tx[1:]
			delete(p.all, firstTx.GetHash())

			// 🌟🌟🌟 关键修复：更新账户nonce
			account, err := p.statedb.Get(from)
			if err == nil {
				account.Nonce = firstTx.Nonce
				p.statedb.Put(from, account)
				fmt.Printf("📝 更新账户nonce: %d → %d\n", account.Nonce-1, account.Nonce)
			}

			if len(box.Tx) == 0 {
				// 盒子空了，从globalBox中删除
				p.globalBox = append(p.globalBox[:i], p.globalBox[i+1:]...)

				// 🌟🌟🌟 修复：从pending中删除空盒子
				if pendingBoxes, exists := p.pending[from]; exists {
					newPendingBoxes := []transaction.Txbox{}

					// 重建pending，移除所有空盒子
					for _, pendingBox := range pendingBoxes {
						if len(pendingBox.Tx) > 0 {
							newPendingBoxes = append(newPendingBoxes, pendingBox)
						}
					}

					if len(newPendingBoxes) == 0 {
						// 没有剩余盒子，删除整个地址条目
						delete(p.pending, from)
						fmt.Printf("🧹 删除地址 %x 的所有pending交易\n", from[:8])
					} else {
						// 更新pending
						p.pending[from] = newPendingBoxes
						fmt.Printf("🗑️ 删除地址 %x 的空盒子，剩余 %d 个盒子\n", from[:8], len(newPendingBoxes))
					}
				}

			} else {
				// 更新盒子
				p.globalBox[i] = box

				// 更新pending中对应的盒子
				if pendingBoxes, exists := p.pending[from]; exists {
					for j, pendingBox := range pendingBoxes {
						// 找到对应的盒子并更新
						if len(pendingBox.Tx) > 0 && len(box.Tx) > 0 &&
							pendingBox.GasPrice == box.GasPrice {
							pendingBoxes[j] = box
							break
						}
					}
					p.pending[from] = pendingBoxes
				}
			}

			// 🌟 弹出交易后检查queue中的交易
			p.promoteQueuedTransactions(from, firstTx.Nonce)

			return firstTx
		}
	}

	// 没有找到可执行的交易
	fmt.Println("没有可执行的交易（nonce不连续）")
	return transaction.Transaction{}
}

// 检查交易是否可以执行
func (p *TxPool) canExecuteTransaction(from common.Address, nonce uint64) bool {
	account, err := p.statedb.Get(from)
	if err != nil {
		fmt.Printf("❌ 获取账户状态失败: %v\n", err)
		return false
	}

	// 检查nonce是否是下一个期望的nonce
	expectedNonce := account.Nonce + 1
	fmt.Printf("🔍 nonce检查: 交易nonce=%d, 账户nonce=%d, 期望nonce=%d\n",
		nonce, account.Nonce, expectedNonce)
	return nonce == expectedNonce
}

func (p *TxPool) SetStateRoot(root common.Hash) {
	// 设置状态根地址
	if err := p.statedb.SetStateRoot(root); err != nil {
		fmt.Printf("Failed to set state root: %v\n", err)
	}
}

func (p *TxPool) addQueueTransaction(tx *transaction.Transaction) error {
	from := tx.From()
	nonce := tx.Nonce
	fmt.Printf("🚀 Adding transaction to queue: Nonce=%d", nonce)
	// 初始化该地址的队列（如果不存在）
	if _, exists := p.queue[from]; !exists {
		p.queue[from] = make(map[uint64]transaction.Transaction)
	}
	// 添加到队列，使用 nonce 作为键
	p.queue[from][nonce] = *tx

	// 不需要排序，map[uint64] 在迭代时会按键排序
	return nil
}

func (p *TxPool) addPendingTransaction(boxes []transaction.Txbox, tx *transaction.Transaction) {
	if len(boxes) == 0 {
		// 如果没有盒子，创建一个新的盒子
		fmt.Printf("🚀 Creating new box for transaction %s\n", tx.GetHash())
		newBox := transaction.Txbox{
			Tx:       []transaction.Transaction{*tx},
			GasPrice: tx.GasPrice,
		}
		p.pending[tx.From()] = []transaction.Txbox{newBox}
		//sort by GasPrice
		fmt.Printf("🌟 Adding new box to global box\n")
		p.globalBox = append(p.globalBox, newBox)
		fmt.Printf("🌟 Global box length: %d\n", len(p.globalBox))
		sort.Sort(&p.globalBox) // 全局盒子排序
	} else {
		lastbox := boxes[len(boxes)-1]
		// 如果最后一个盒子的GasPrice小于等于当前交易的GasPrice
		if lastbox.GasPrice <= tx.GasPrice {
			// 将交易添加到最后一个盒子中
			fmt.Printf("🌟 Adding transaction to last box with GasPrice %d\n", lastbox.GasPrice)
			lastbox.Tx = append(lastbox.Tx, *tx)

			// 重要修改: 更新盒子的GasPrice（如果新交易GasPrice更高）
			if tx.GasPrice > lastbox.GasPrice {
				fmt.Printf("🔄 Updating box GasPrice from %d to %d\n", lastbox.GasPrice, tx.GasPrice)
				lastbox.GasPrice = tx.GasPrice
			}

			// 更新最后一个盒子
			p.pending[tx.From()][len(p.pending[tx.From()])-1] = lastbox

			// 同步更新全局盒子
			for i, globalBox := range p.globalBox {
				// 找到对应的全局盒子（通过比较盒子中交易的哈希）
				if len(globalBox.Tx) > 0 && len(lastbox.Tx) > 0 {
					// 可以通过比较盒子中第一个交易的哈希来确定是否是同一个盒子
					if globalBox.Tx[0].GetHash() == lastbox.Tx[0].GetHash() {
						p.globalBox[i] = lastbox
						break
					}
				}
			}

			fmt.Printf("🌟 Updated last box: GasPrice=%d, 交易数=%d,地址=%p\n", lastbox.GasPrice, len(lastbox.Tx), &lastbox.Tx)

			// 重要: 重新排序全局盒子
			sort.Sort(&p.globalBox)
		} else {
			// 如果最后一个盒子的GasPrice大于当前交易的GasPrice
			fmt.Printf("🚀 Creating new box for transaction with GasPrice %d\n", tx.GasPrice)
			// 创建一个新的盒子，并将交易添加到新的盒子中
			newBox := transaction.Txbox{
				Tx:       []transaction.Transaction{*tx},
				GasPrice: tx.GasPrice,
			}
			// 将新的盒子添加到pending中
			p.pending[tx.From()] = append(p.pending[tx.From()], newBox)
			// 将新的盒子添加到全局盒子中
			p.globalBox = append(p.globalBox, newBox)
			//sort by GasPrice
			sort.Sort(&p.globalBox)
		}
	}
	// 自动提升queue中的后续交易
	p.promoteQueuedTransactions(tx.From(), tx.Nonce)
	return
}

func (p *TxPool) promoteQueuedTransactions(from common.Address, lastNonce uint64) {
	queuedTxs, exists := p.queue[from]
	if !exists || len(queuedTxs) == 0 {
		return
	}

	// 从lastNonce+1开始，连续提升交易
	nextNonce := lastNonce + 1

	for {
		tx, exists := queuedTxs[nextNonce]
		if !exists {
			break // 没有连续的nonce，停止提升
		}

		fmt.Printf("✅ 提升交易: Nonce=%d, GasPrice=%d\n", tx.Nonce, tx.GasPrice)

		// 从队列中删除
		delete(queuedTxs, nextNonce)

		// 添加到pending
		currentPending := p.pending[from]
		p.addPendingTransaction(currentPending, &tx)

		nextNonce++
	}

	// 清理空的队列
	if len(queuedTxs) == 0 {
		delete(p.queue, from)
	}
}

func (p *TxPool) replace(boxes []transaction.Txbox, tx *transaction.Transaction) error {
	for i, box := range boxes {
		for j, existingTx := range box.Tx {
			if existingTx.Nonce == tx.Nonce {
				// 检查GasPrice
				if tx.GasPrice <= existingTx.GasPrice {
					return fmt.Errorf("replacement gas price %d not higher than existing %d",
						tx.GasPrice, existingTx.GasPrice)
				}

				// ✅ 拆分盒子方案
				err := p.replaceWithSplit(boxes, i, j, tx, existingTx)
				if err != nil {
					return err
				}

				// 🌟 替换后检查queue中的交易
				p.promoteQueuedTransactions(tx.From(), tx.Nonce)

				return nil
			}
		}
	}
	return errors.New("transaction not found for replacement")
}

func (p *TxPool) replaceWithSplit(boxes []transaction.Txbox, boxIndex, txIndex int, newTx *transaction.Transaction, oldTx transaction.Transaction) error {
	box := boxes[boxIndex]
	originalTxs := box.Tx

	// 拆分为三部分
	beforeTxs := originalTxs[:txIndex]  // 替换交易之前的交易
	afterTxs := originalTxs[txIndex+1:] // 曦换交易之后的交易

	// 从原boxes中移除要拆分的盒子
	newBoxes := append(boxes[:boxIndex], boxes[boxIndex+1:]...)

	// 创建新盒子
	createdBoxes := []transaction.Txbox{}

	// 1. 前部分交易的盒子
	if len(beforeTxs) > 0 {
		beforeBox := transaction.Txbox{
			Tx:       beforeTxs,
			GasPrice: p.calculateMaxGasPrice(beforeTxs), // 使用最高的GasPrice
		}
		createdBoxes = append(createdBoxes, beforeBox)
	}

	// 2. 新交易的独立盒子
	newTxBox := transaction.Txbox{
		Tx:       []transaction.Transaction{*newTx},
		GasPrice: newTx.GasPrice,
	}
	createdBoxes = append(createdBoxes, newTxBox)

	// 3. 后部分交易的盒子
	if len(afterTxs) > 0 {
		afterBox := transaction.Txbox{
			Tx:       afterTxs,
			GasPrice: p.calculateMaxGasPrice(afterTxs),
		}
		createdBoxes = append(createdBoxes, afterBox)
	}

	// 合并所有盒子并按GasPrice排序
	allBoxes := append(newBoxes, createdBoxes...)
	sort.Slice(allBoxes, func(i, j int) bool {
		return allBoxes[i].GasPrice > allBoxes[j].GasPrice
	})

	// 更新pending和globalBox
	p.pending[newTx.From()] = allBoxes
	p.all[newTx.GetHash()] = true
	delete(p.all, oldTx.GetHash())

	p.rebuildGlobalBoxForAddress(newTx.From())

	return nil
}

func (p *TxPool) calculateMaxGasPrice(txs []transaction.Transaction) uint64 {
	if len(txs) == 0 {
		return 0
	}

	maxGasPrice := txs[0].GasPrice
	for _, tx := range txs {
		if tx.GasPrice > maxGasPrice {
			maxGasPrice = tx.GasPrice
		}
	}
	return maxGasPrice
}

// 重建特定地址的globalBox条目
func (p *TxPool) rebuildGlobalBoxForAddress(addr common.Address) {
	// 从globalBox中移除该地址的所有盒子
	newGlobalBox := transaction.Boxes{}
	for _, globalBox := range p.globalBox {
		if len(globalBox.Tx) > 0 && globalBox.Tx[0].From() != addr {
			newGlobalBox = append(newGlobalBox, globalBox)
		}
	}

	// 重新添加该地址的所有盒子
	if boxes, exists := p.pending[addr]; exists {
		for _, box := range boxes {
			if len(box.Tx) > 0 {
				newGlobalBox = append(newGlobalBox, box)
			}
		}
	}

	// 更新并重新排序
	p.globalBox = newGlobalBox
	sort.Sort(&p.globalBox)
}


