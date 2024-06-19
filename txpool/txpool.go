package txpool

import (
	"cxchain223/types"
	"cxchain223/utils/hash"
)

type TxPool interface {
	SetStatRoot(root hash.Hash)
	NewTx(tx *types.Transaction)
	Pop() *types.Transaction
	NotifyTxEvent(txs []*types.Transaction)
}
