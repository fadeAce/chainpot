package chainpot

import (
	"context"
	"errors"
	"fmt"
	"github.com/fadeAce/claws"
	"github.com/fadeAce/claws/types"
	"math/big"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
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
	Content *BlockMessage
}

type Chain struct {
	*sync.Mutex
	addrs           map[string]bool
	currentBlocks   map[int64]int64
	config          *chainOption
	height          int64
	handledEndPoint bool
	wallet          claws.Wallet
	depositTxs      *Queue
	withdrawTxs     *Queue
	storage         *storage
	onMessage       func(msg *PotEvent)
	ctx             context.Context
	cancel          context.CancelFunc
}

type chainOption struct {
	LogPath      string
	ConfirmTimes int64
	Chain        string
	IDX          int
	Endpoint     int64
}

func newChain(opt *chainOption, wallet claws.Wallet) *Chain {
	ctx, cancel := context.WithCancel(context.Background())
	chain := &Chain{
		Mutex:         &sync.Mutex{},
		addrs:         make(map[string]bool),
		currentBlocks: make(map[int64]int64),
		config:        opt,
		wallet:        wallet,
		depositTxs:    NewQueue(),
		withdrawTxs:   NewQueue(),
		storage:       newStorage(opt.LogPath, opt.Chain),
		ctx:           ctx,
		cancel:        cancel,
	}

	var fp = opt.LogPath + "/" + opt.Chain
	if _, err := os.Stat(fp); err != nil {
		os.Mkdir(fp, 0755)
	}

	return chain
}

func (c *Chain) start() {
	var notice = make(chan int64)
	go func() {
		c.wallet.NotifyHead(c.ctx, func(num *big.Int) {
			notice <- num.Int64()
		})
	}()

	go func() {
		var ticker = time.NewTicker(180 * time.Second)
		defer ticker.Stop()

		select {
		case <-c.ctx.Done():
			println(fmt.Sprintf("%s stopped", c.config.Chain))
			return
		case <-ticker.C:
			var now = time.Now().UnixNano() / 1000000
			for k, v := range c.currentBlocks {
				if now-v > 180000 {
					delete(c.currentBlocks, k)
				}
			}
		case height := <-notice:
			if height > c.height {
				atomic.StoreInt64(&c.height, height)
			}
			println(fmt.Sprintf("Synchronizing Block: %d", height))
			c.handleEndpoint(c.config.Endpoint, height)
			c.handleBlock(big.NewInt(height), false)
		}
	}()
}

func (c *Chain) handleBlock(num *big.Int, useCache bool) {
	var height = num.Int64()
	if _, exist := c.currentBlocks[height]; exist {
		return
	}

	var txns = make([]types.TXN, 0)
	var err error
	if useCache {
		var block, e = c.storage.getBlock(height)
		if e != nil {
			for i, _ := range block {
				txns = append(txns, block[i])
			}
		}
		err = e
	}
	if !useCache || err != nil {
		txns, err = c.wallet.UnfoldTxs(context.Background(), num)
	}
	if err != nil {
		return
	}

	c.currentBlocks[height] = time.Now().UnixNano() / 1000000
	var block = make([]*BlockMessage, 0)
	for i, _ := range txns {
		var tx = txns[i]
		var _, f1 = c.addrs[tx.FromStr()]
		var _, f2 = c.addrs[tx.ToStr()]
		var node = &Value{TXN: tx, Height: height, Index: int64(i)}
		if (f1 || f2) && tx.FromStr() == tx.ToStr() {
			c.onMessage(&PotEvent{
				Chain: c.config.Chain,
				ID:    getEventID(height, T_ERROR, int64(i)),
				Event: T_ERROR,
			})
		} else if f1 && f2 {
			c.withdrawTxs.PushBack(node)
			var cp = *node
			c.depositTxs.PushBack(&cp)
		} else if f1 {
			c.withdrawTxs.PushBack(node)
		} else if f2 {
			c.depositTxs.PushBack(node)
		}

		if f1 || f2 {
			block = append(block, NewBlockMessage(tx))
		}
	}

	if !useCache {
		c.storage.saveBlock(height, block)
	}
	c.emitter()
}

func (c *Chain) handleEndpoint(endpoint int64, currentHeight int64) {
	if c.handledEndPoint || c.config.Endpoint <= 0 {
		return
	}

	for i := endpoint; i < currentHeight; i++ {
		c.handleBlock(big.NewInt(i), true)
	}
	c.handledEndPoint = true
}

// emit events
func (c *Chain) emitter() {
	var m = c.depositTxs.Len()
	for i := 0; i < m; i++ {
		var val = c.depositTxs.Front()
		var msg = &PotEvent{
			Chain:   c.config.Chain,
			Content: NewBlockMessage(val.TXN),
		}

		//var tx = types.TXN(inter).SetStr()
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
		c.onMessage(msg)
	}

	var n = c.withdrawTxs.Len()
	for i := 0; i < n; i++ {
		var val = c.withdrawTxs.Front()
		var msg = &PotEvent{
			Chain:   c.config.Chain,
			Content: NewBlockMessage(val.TXN),
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
		c.onMessage(msg)
	}
}

func (c *Chain) add(addrs []string) (height int64, err error) {
	c.Lock()
	defer c.Unlock()
	for i, _ := range addrs {
		addr := addrs[i]
		if _, exist := c.addrs[addr]; exist {
			return 0, errors.New("repeat add")
		}
		c.addrs[addr] = true
	}
	return c.height, nil
}

func getEventID(height int64, event EventType, idx int64) int64 {
	var str = fmt.Sprintf("%d%03d%06d", height, event, idx)
	num, _ := strconv.Atoi(str)
	return int64(num)
}
