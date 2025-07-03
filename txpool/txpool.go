package txpool

import (
	"chain/common"
	"chain/transaction"
	"chain/statedb"
)

type Pool interface {
	// addTransaction is a function field for adding a transaction
	AddTransaction (tx *transaction.Transaction) 
	// PopTransaction is a function field for popping a transaction
	PopTransaction ()transaction.Transaction
	// SetStateRoot is a function field for setting the state root
	SetStateRoot(root common.Hash) 
	// NotifyTrascation is a function field for notifying transactions
	NotifyTrascation(txs []*transaction.Transaction)
	//create a new transaction pool
	NewTxPool(statedb statedb.StateDB) *Pool
}