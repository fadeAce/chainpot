package chainpot

import (
	"context"
	"github.com/fadeAce/claws"
	"github.com/fadeAce/claws/types"
	"math/big"
	"time"
)

type MaskBuilder struct{}

func (c *MaskBuilder) Build() claws.Wallet {
	return &MaskWallet{}
}

type MaskWallet struct{}

func (c *MaskWallet) Type() string {
	return "MLGB"
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
	return []types.TXN{&BlockMessage{
		Hash:   "0xda29054d35d1af9d54e5e8aafce62fec11c716c8bef67508e2dc4ae5e3882ebb",
		From:   "0x78aE889cd04Cb9274C2600d68CCc5058F43dB63e",
		To:     "0x54a298ee9fccbf0ad8e55bc641d3086b81a48c41",
		Fee:    "0.000247385",
		Amount: "0.01",
	}}, nil
}

func (c *MaskWallet) NotifyHead(ctx context.Context, f func(num *big.Int)) error {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		var idx int64 = 8
		defer ticker.Stop()
		for {
			<-ticker.C
			f(big.NewInt(idx))
			idx++
		}
	}()
	return nil
}

func (c *MaskWallet) Info() *types.Info {
	return &types.Info{}
}
