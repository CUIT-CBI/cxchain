package txpool

import (
	"cxchain221/statdb"
	"cxchain221/types"
	"cxchain221/utils/hash"
	"sort"
)

type DefaultPool struct {
	StatDB  statdb.StatDB
	all     map[hash.Hash]bool
	txs     PendingTxs
	pending map[types.Address]PendingTxs
	queue   map[types.Address]SortedTxs
}

type SubPool interface {
	Push(tx *types.Transaction)
	Pop() *types.Transaction
	GasPrice() uint64
	Nonce() uint64
	Address() types.Address
	Replace(tx *types.Transaction)
}

type SortedTxs []*types.Transaction

func (txs SortedTxs) Push(tx *types.Transaction) {
	// todo
	txs = append(txs, tx)
}

type PendingTxs []SubPool

func (txs PendingTxs) Len() int {
	return len(txs)
}

func (txs PendingTxs) Less(i, j int) bool {
	if txs[i].Address() == txs[j].Address() {
		return txs[i].Nonce() > txs[j].Nonce()
	}
	return txs[i].GasPrice() < txs[j].GasPrice()
}

func (txs PendingTxs) Swap(i, j int) {
	txs[i], txs[j] = txs[j], txs[i]
}

func (pool DefaultPool) NewTx(tx *types.Transaction) {
	account := pool.StatDB.Load(tx.From())
	if account.Nonce >= tx.Nonce {
		return
	}

	nonce := account.Nonce
	pools := pool.pending[tx.From()]
	if len(pools) > 0 {
		last := pools[len(pools)-1]
		nonce = last.Nonce()
	}
	if tx.Nonce > nonce+1 {
		// 加到queue
		pool.addQueueTx(tx)
	} else if tx.Nonce == nonce+1 {
		// 加到pending，判断是否有queue的交易可以pop
		pool.addPendingTx(tx)
	} else {
		// replace
		pool.replacePendingTx(tx)
	}
}

func (pool DefaultPool) addQueueTx(tx *types.Transaction) {
	txs := pool.queue[tx.From()]
	txs = append(txs, tx)
	// TODO 对txs进行排序
	pool.queue[tx.From()] = txs
}

func (pool DefaultPool) addPendingTx(tx *types.Transaction) {
	subpools := pool.pending[tx.From()]
	if len(subpools) == 0 {
		sub := make(SortedTxs, 1)
		sub = append(sub, tx)
		subpools = append(subpools, sub)
		pool.txs = append(pool.txs, sub)
		sort.Sort(pool.txs)
	} else {
		last := subpools[len(subpools)-1]
		if last.GasPrice() <= tx.GasPrice {
			last = append(last, tx)
		} else {
			sub := make(SortedTxs, 1)
			sub = append(sub, tx)
			subpools = append(subpools, sub)
			pool.txs = append(pool.txs, sub)
			sort.Sort(pool.txs)
		}
	}
	// TODO 更新queue中可以pop到pending中的交易
}

func (pool DefaultPool) replacePendingTx(tx *types.Transaction) {
	subpools := pool.pending[tx.From()]
	for _, sub := range subpools {
		if sub.Nonce() >= tx.Nonce {
			if tx.GasPrice >= sub.GasPrice() {
				sub.Replace(tx)
			}
			break
		}
	}
}

func (pool DefaultPool) Nonce(addr types.Address) uint64 {

}

func (pool DefaultPool) Pop() *types.Transaction {}

func (pool DefaultPool) SetStatRoot(root hash.Hash) {}

func (pool DefaultPool) NotifyTxEvent(txs []*types.Transaction) {}
