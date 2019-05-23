package chainpot

import (
	"context"
	"github.com/fadeAce/claws"
	"math/big"
	"sync"
	"sync/atomic"
)

type EventType int

const (
	// NORMAL STATE
	T_DEPOSIT EventType = iota
	T_WITHDRAW
	T_DEPOSIT_UPDATE
	T_WITHDRAW_UPDATE
	T_WITHDRAW_CONFIRM
	T_DEPOSIT_CONFIRM

	// ABNORMAL STATE
	T_WITHDRAW_FAIL
)

type PotEvent struct {
	Chain   string
	Event   EventType
	Content interface{}
}

type BlockMessage struct {
	Hash   string
	From   string
	To     string
	Fee    string
	Amount string
}

type Filter func(msg []BlockMessage) []BlockMessage

type Chainpot struct {
	*sync.RWMutex
	ID           int64
	Code         string
	addrs        map[string]bool
	messenger    chan []BlockMessage
	config       *ChainOption
	isRegistered bool
	height       int64
	wallet       claws.Wallet
}

type ChainOption struct {
	ConfirmTimes int64
	Code         string
}

func NewChainpot(opt *ChainOption, wallet claws.Wallet) *Chainpot {
	chain := &Chainpot{
		RWMutex:   &sync.RWMutex{},
		addrs:     make(map[string]bool),
		messenger: make(chan []BlockMessage, 1024),
		config:    opt,
		wallet:    wallet,
	}
	chain.listen()
	return chain
}

func (c *Chainpot) listen() {
	c.wallet.NotifyHead(context.Background(), func(num *big.Int) {
		txns, err := c.wallet.UnfoldTxs(context.Background(), num)
		if err != nil {
			return
		}

		// TODO check multi threads
		if num.Int64() > c.height {
			atomic.StoreInt64(&c.height, num.Int64())
		}

		var depMsgs = make([]*BlockMessage, 0)
		var witMsgs = make([]*BlockMessage, 0)
		var errMsgs = make([]*BlockMessage, 0)
		c.RLock()
		for i, _ := range txns {
			var tx = txns[i]
			var msg = &BlockMessage{
				Hash:   tx.HexStr(),
				Amount: tx.AmountStr(),
				From:   tx.FromStr(),
				To:     tx.ToStr(),
				Fee:    tx.FeeStr(),
			}

			var _, f1 = c.addrs[msg.From]
			var _, f2 = c.addrs[msg.To]
			if f1 && f2 {
				errMsgs = append(errMsgs, msg)
			} else if f1 {
				witMsgs = append(witMsgs, msg)
			} else if f2 {
				depMsgs = append(depMsgs, msg)
			}
		}
		c.RUnlock()

		for _, msg := range depMsgs {

		}
	})
}

func (c *Chainpot) Add(addrs []string) {
	c.Lock()
	for i, _ := range addrs {
		addr := addrs[i]
		c.addrs[addr] = true
	}
	c.Unlock()
}

func (c *Chainpot) OnMessage(cb func(msgs []BlockMessage)) {
	go func() {
		for {
			msgs := <-c.messenger
			cb(msgs)
		}
	}()
}
