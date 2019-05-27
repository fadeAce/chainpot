package chainpot

import (
	"context"
	"github.com/fadeAce/chainpot/queue"
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
	T_ERROR
)

type PotEvent struct {
	Chain   string
	Event   EventType
	Content types.TXN
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
	*sync.Mutex
	ID          int64
	addrs       map[string]bool
	messenger   chan *PotEvent
	config      *ChainOption
	height      int64
	wallet      claws.Wallet
	depositTxs  *queue.Queue
	withdrawTxs *queue.Queue
	OnMessage   func(msg *PotEvent)
}

type ChainOption struct {
	ConfirmTimes int64
	Chain        string
}

func NewChainpot(opt *ChainOption, wallet claws.Wallet) *Chainpot {
	chain := &Chainpot{
		Mutex:       &sync.Mutex{},
		addrs:       make(map[string]bool),
		messenger:   make(chan *PotEvent, 1024),
		config:      opt,
		wallet:      wallet,
		depositTxs:  queue.NewQueue(1024),
		withdrawTxs: queue.NewQueue(1024),
	}
	go chain.listener()
	return chain
}

func (c *Chainpot) listener() {
	c.wallet.NotifyHead(context.Background(), func(num *big.Int) {
		var height = num.Int64()
		txns, err := c.wallet.UnfoldTxs(context.Background(), num)
		if err != nil {
			return
		}

		// TODO check multi threads
		if num.Int64() > c.height {
			atomic.StoreInt64(&c.height, height)
		}

		var confirmTimes = c.height - height + 1
		if confirmTimes > c.config.ConfirmTimes {
			return
		}

		c.Lock()
		for i, _ := range txns {
			var tx = txns[i]
			var _, f1 = c.addrs[tx.FromStr()]
			var _, f2 = c.addrs[tx.ToStr()]
			var node = &queue.Value{TXN: tx, Height: height}
			if (f1 || f2) && tx.FromStr() == tx.ToStr() {
				c.OnMessage(&PotEvent{
					Chain: c.config.Chain,
					Event: T_ERROR,
				})
			} else if f1 && f2 {
				c.withdrawTxs.PushBack(node)
				c.depositTxs.PushBack(node)
			} else if f1 {
				c.withdrawTxs.PushBack(node)
			} else if f2 {
				c.depositTxs.PushBack(node)
			}
		}
		c.emitter()
		c.Unlock()
	})
}

// emit events
func (c *Chainpot) emitter() {
	var m = c.depositTxs.Len()
	for i := 0; i < m; i++ {
		var val = c.depositTxs.Front()
		var msg = &PotEvent{
			Chain:   c.config.Chain,
			Content: val.TXN,
		}

		if !c.wallet.Seek(val.TXN) {
			continue
		}

		if c.height-val.Height+1 >= c.config.ConfirmTimes {
			msg.Event = T_DEPOSIT_CONFIRM
		} else if c.height-val.Height == 0 {
			msg.Event = T_DEPOSIT
			c.depositTxs.PushBack(val)
		} else {
			msg.Event = T_DEPOSIT_UPDATE
			c.depositTxs.PushBack(val)
		}
		c.OnMessage(msg)
	}

	var n = c.withdrawTxs.Len()
	for i := 0; i < n; i++ {
		var val = c.withdrawTxs.Front()
		var msg = &PotEvent{
			Chain:   c.config.Chain,
			Content: val.TXN,
		}

		if !c.wallet.Seek(val.TXN) {
			msg.Event = T_WITHDRAW_FAIL
		} else if c.height-val.Height+1 >= c.config.ConfirmTimes {
			msg.Event = T_WITHDRAW_CONFIRM
		} else if c.height-val.Height == 0 {
			msg.Event = T_WITHDRAW
			c.withdrawTxs.PushBack(val)
		} else {
			msg.Event = T_WITHDRAW_UPDATE
			c.withdrawTxs.PushBack(val)
		}
		c.OnMessage(msg)
	}
}

func (c *Chainpot) Add(addrs []string) {
	c.Lock()
	for i, _ := range addrs {
		addr := addrs[i]
		c.addrs[addr] = true
	}
	c.Unlock()
}
