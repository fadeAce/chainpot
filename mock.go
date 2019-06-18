package chainpot

import (
	"context"
	"github.com/fadeAce/claws"
	"github.com/fadeAce/claws/types"
	"math/big"
	"time"
)

var (
	testMsgs   map[int64]*[]BlockMessage
	testHeight = 1
)

type MaskBuilder struct{}

func (c *MaskBuilder) Build() claws.Wallet {
	testMsgs = make(map[int64]*[]BlockMessage)
	return &MaskWallet{}
}

type MaskWallet struct{}

func (c *MaskWallet) Type() string {
	return "mlgb"
}

func (c *MaskWallet) InitWallet() {

}

func (c *MaskWallet) NewAddr() types.Bundle {
	return types.Bundle(nil)
}

func (c *MaskWallet) BuildBundle(prv, pub, addr string) types.Bundle {
	return types.Bundle(nil)
}

func (c *MaskWallet) BuildTxn(hash string) types.TXN {
	return &BlockMessage{Hash: hash}
}

func (c *MaskWallet) Withdraw(addr types.Bundle) *types.TxnInfo {
	return &types.TxnInfo{}
}

func (c *MaskWallet) Seek(txn types.TXN) bool {
	return true
}

func (c *MaskWallet) Balance(bundle types.Bundle) (string, error) {
	return "10000", nil
}

func (c *MaskWallet) UnfoldTxs(ctx context.Context, num *big.Int) ([]types.TXN, error) {
	height := num.Int64()
	arr, ok := testMsgs[height]
	if !ok {
		return make([]types.TXN, 0), nil
	}

	txns := make([]types.TXN, 0)
	for _, item := range *arr {
		txns = append(txns, &item)
	}
	return txns, nil
}

func (c *MaskWallet) NotifyHead(ctx context.Context, f func(num *big.Int)) error {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		var height int64 = 8
		for {
			select {
			case <-ticker.C:
				f(big.NewInt(height))
				height++
			}
		}
	}()
	return nil
}

func (c *MaskWallet) Send(ctx context.Context, from, to types.Bundle, amount string, option *types.Option) (err error) {
	return nil
}

func pushBack(height int64, tx BlockMessage) {
	arr, ok := testMsgs[height]
	if !ok {
		list := make([]BlockMessage, 0)
		testMsgs[height] = &list
		arr = &list
	}
	*arr = append(*arr, tx)
}

func (c *MaskWallet) Info() *types.Info {
	return &types.Info{}
}
