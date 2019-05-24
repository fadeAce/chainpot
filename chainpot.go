package chainpot

import (
	"context"
	"github.com/fadeAce/claws"
	"github.com/fadeAce/claws/types"
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
	Content BlockMessage
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
	ID        int64
	addrs     map[string]bool
	messenger chan *PotEvent
	config    *ChainOption
	height    int64
	wallet    claws.Wallet
}

type ChainOption struct {
	ConfirmTimes int64
	Chain        string
}

func NewChainpot(opt *ChainOption, wallet claws.Wallet) *Chainpot {
	chain := &Chainpot{
		RWMutex:   &sync.RWMutex{},
		addrs:     make(map[string]bool),
		messenger: make(chan *PotEvent, 1024),
		config:    opt,
		wallet:    wallet,
	}
	go chain.listen()
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

		var confirmTimes = c.height - num.Int64() + 1
		if confirmTimes > c.config.ConfirmTimes {
			return
		}

		var depTxs = make([]types.TXN, 0)
		var witTxs = make([]types.TXN, 0)
		var errTxs = make([]types.TXN, 0)
		c.RLock()
		for i, _ := range txns {
			var tx = txns[i]

			var _, f1 = c.addrs[tx.FromStr()]
			var _, f2 = c.addrs[tx.ToStr()]
			if f1 && f2 {
				errTxs = append(errTxs, tx)
			} else if f1 {
				witTxs = append(witTxs, tx)
			} else if f2 {
				depTxs = append(depTxs, tx)
			}
		}
		c.RUnlock()

		for _, tx := range depTxs {
			var msg = &PotEvent{
				Chain: c.config.Chain,
				Content: BlockMessage{
					Hash:   tx.HexStr(),
					Amount: tx.AmountStr(),
					From:   tx.FromStr(),
					To:     tx.ToStr(),
					Fee:    tx.FeeStr(),
				},
			}
			if confirmTimes == 1 {
				msg.Event = T_DEPOSIT
			} else if confirmTimes == c.config.ConfirmTimes {
				msg.Event = T_DEPOSIT_CONFIRM
			} else {
				if c.wallet.Seek(tx) {
					msg.Event = T_DEPOSIT_UPDATE
				}
			}
			c.messenger <- msg
		}

		for _, tx := range witTxs {
			var msg = &PotEvent{
				Chain: c.config.Chain,
				Content: BlockMessage{
					Hash:   tx.HexStr(),
					Amount: tx.AmountStr(),
					From:   tx.FromStr(),
					To:     tx.ToStr(),
					Fee:    tx.FeeStr(),
				},
			}
			if confirmTimes == 1 {
				msg.Event = T_WITHDRAW
			} else if confirmTimes == c.config.ConfirmTimes {
				msg.Event = T_WITHDRAW_CONFIRM
			} else {
				if c.wallet.Seek(tx) {
					msg.Event = T_WITHDRAW_UPDATE
				}
			}
			c.messenger <- msg
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

func (c *Chainpot) OnMessage(cb func(msg *PotEvent)) {
	func() {
		for {
			msg := <-c.messenger
			cb(msg)
		}
	}()
}
