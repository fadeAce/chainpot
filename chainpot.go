package chainpot

import (
	"context"
	"fmt"
	"github.com/fadeAce/chainpot/queue"
	"github.com/fadeAce/claws"
	"github.com/fadeAce/claws/types"
	"math/big"
	"strconv"
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
	ID      int64
	Content types.TXN
}

type Chainpot struct {
	*sync.Mutex
	addrs           map[string]bool
	config          *ChainOption
	height          int64
	handledEndPoint bool
	wallet          claws.Wallet
	depositTxs      *queue.Queue
	withdrawTxs     *queue.Queue
	OnMessage       func(msg *PotEvent)
}

type ChainOption struct {
	ConfirmTimes int64
	Chain        string
	Endpoint     int64
}

func NewChainpot(opt *ChainOption, wallet claws.Wallet) *Chainpot {
	chain := &Chainpot{
		Mutex:       &sync.Mutex{},
		addrs:       make(map[string]bool),
		config:      opt,
		wallet:      wallet,
		depositTxs:  queue.NewQueue(1024),
		withdrawTxs: queue.NewQueue(1024),
	}
	go func() {
		chain.wallet.NotifyHead(context.Background(), func(num *big.Int) {
			if num.Int64() > chain.height {
				atomic.StoreInt64(&chain.height, num.Int64())
			}
			chain.handleEndpoint(chain.config.Endpoint, num.Int64())
			chain.handleBlock(num)
		})
	}()
	return chain
}

func (c *Chainpot) handleBlock(num *big.Int) {
	var height = num.Int64()
	txns, err := c.wallet.UnfoldTxs(context.Background(), num)
	if err != nil {
		return
	}

	c.Lock()
	for i, _ := range txns {
		var tx = txns[i]
		var _, f1 = c.addrs[tx.FromStr()]
		var _, f2 = c.addrs[tx.ToStr()]
		var node = &queue.Value{TXN: tx, Height: height, Index: int64(i)}
		if (f1 || f2) && tx.FromStr() == tx.ToStr() {
			c.OnMessage(&PotEvent{
				Chain: c.config.Chain,
				ID:    getEventID(height, T_ERROR, int64(i)),
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
}

func (c *Chainpot) handleEndpoint(lastHeight int64, latestHeight int64) {
	if c.handledEndPoint || c.config.Endpoint == 0 {
		return
	}

	for i := lastHeight; i < latestHeight; i++ {
		c.handleBlock(big.NewInt(i))
	}
	c.handledEndPoint = true
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
		msg.ID = getEventID(val.Height, msg.Event, val.Index)
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
		msg.ID = getEventID(val.Height, msg.Event, val.Index)
		c.OnMessage(msg)
	}
}

func (c *Chainpot) Add(addrs []string) (height int64) {
	c.Lock()
	for i, _ := range addrs {
		addr := addrs[i]
		c.addrs[addr] = true
	}
	c.Unlock()
	return c.height
}

func getEventID(height int64, event EventType, idx int64) int64 {
	var str = fmt.Sprintf("%d%03d%06d", height, event, idx)
	num, _ := strconv.Atoi(str)
	return int64(num)
}
