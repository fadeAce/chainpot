package chainpot

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/fadeAce/claws"
	"github.com/rs/zerolog/log"
	"math/big"
	"os"
	"strings"
	"sync"
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
	ID      string
	Content *BlockMessage
}

type Chain struct {
	*sync.Mutex
	addrs          map[string]bool
	syncedTxs      map[string]int64
	config         *CoinConf
	height         int64
	syncedEndPoint bool
	wallet         claws.Wallet
	depositTxs     *Queue
	withdrawTxs    *Queue
	storage        *storage
	noticer        chan *big.Int
	onMessage      func(msg *PotEvent)
	ctx            context.Context
	cancel         context.CancelFunc
}

func newChain(opt *CoinConf, wallet claws.Wallet) *Chain {
	ctx, cancel := context.WithCancel(context.Background())
	stg := newStorage(opt.Code)
	cache, addrs := getCacheConfig(opt.Code)
	opt.Endpoint = cache.EndPoint

	chain := &Chain{
		Mutex:       &sync.Mutex{},
		addrs:       addrs,
		height:      opt.Endpoint,
		syncedTxs:   make(map[string]int64),
		config:      opt,
		wallet:      wallet,
		depositTxs:  NewQueue(),
		withdrawTxs: NewQueue(),
		storage:     stg,
		noticer:     make(chan *big.Int, 1024),
		ctx:         ctx,
		cancel:      cancel,
	}

	var fp = cachePath + "/" + opt.Code
	if _, err := os.Stat(fp); err != nil {
		os.Mkdir(fp, 0755)
	}

	return chain
}

func (c *Chain) start() {
	log.Info().Msgf("%s start", strings.ToUpper(c.config.Code))

	go func() {
		err := c.wallet.NotifyHead(c.ctx, func(num *big.Int) {
			c.noticer <- num
			log.Info().Msgf("%s received new block from claws ", num.String())
			saveCacheConfig(c.config.Code, &cacheConfig{EndPoint: num.Int64()}, nil)
		})
		if err != nil {
			log.Error().Msgf("fatal error when starting head syncing: %s", err.Error())
		}
	}()

	go func() {
		var ticker = time.NewTicker(180 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-c.ctx.Done():
				saveCacheConfig(c.config.Code, &cacheConfig{EndPoint: c.height}, c.addrs)
				wg.Done()
				log.Info().Msgf("%s stopped, endpoint: %d", strings.ToUpper(c.config.Code), c.height)
				return
			case <-ticker.C:
				var now = time.Now().UnixNano() / 1000000
				for k, v := range c.syncedTxs {
					if now-v > 180000 {
						delete(c.syncedTxs, k)
					}
				}
			case num := <-c.noticer:
				height := num.Int64()
				if height > c.height {
					c.height = height
				}
				c.syncEndpoint(c.config.Endpoint, height)
				c.syncBlock(num)
			}
		}
	}()
}

func (c *Chain) syncBlock(num *big.Int) {
	var height = num.Int64()
	log.Info().Msgf("%s Synchronizing Block: %d", strings.ToUpper(c.config.Code), height)
	txns, err := c.wallet.UnfoldTxs(context.Background(), num)
	if err != nil {
		return
	}

	var block = make([]*BlockMessage, 0)
	for i, _ := range txns {
		var tx = txns[i]
		var _, f1 = c.addrs[tx.FromStr()]
		var _, f2 = c.addrs[tx.ToStr()]
		if !f1 && !f2 {
			continue
		}
		if _, exist := c.syncedTxs[tx.HexStr()]; exist {
			continue
		}

		block = append(block, NewBlockMessage(tx))
		c.syncedTxs[tx.HexStr()] = time.Now().UnixNano() / 1000000
		var node = &Value{TXN: tx, Height: height, Index: int64(i)}
		if tx.FromStr() == tx.ToStr() {
			c.onMessage(&PotEvent{
				Chain: c.config.Code,
				ID:    c.getEventID(height, height, T_ERROR, int64(i)),
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
	}

	c.storage.saveBlock(height, block)
	c.emitter()
}

func (c *Chain) syncEndpoint(endpoint int64, currentHeight int64) {
	if c.syncedEndPoint || c.config.Endpoint <= 0 {
		return
	}

	for i := endpoint - c.config.ConfirmTimes; i < currentHeight; i++ {
		c.syncBlock(big.NewInt(i))
	}
	c.syncedEndPoint = true
}

// emit events
func (c *Chain) emitter() {
	var m = c.depositTxs.Len()
	for i := 0; i < m; i++ {
		var val = c.depositTxs.Front()
		var msg = &PotEvent{
			Chain:   c.config.Code,
			Content: NewBlockMessage(val.TXN),
		}

		if c.height-val.Height+1 >= c.config.ConfirmTimes {
			msg.Event = T_DEPOSIT_CONFIRM
			if !c.wallet.Seek(val.TXN) {
				continue
			}
		} else if c.height-val.Height == 0 {
			msg.Event = T_DEPOSIT
			c.depositTxs.PushBack(val)
		} else {
			msg.Event = T_DEPOSIT_UPDATE
			c.depositTxs.PushBack(val)
		}
		msg.ID = c.getEventID(c.height, val.Height, msg.Event, val.Index)
		log.Debug().Msgf("New Event: %s", mustMarshal(msg))
		c.onMessage(msg)
	}

	var n = c.withdrawTxs.Len()
	for i := 0; i < n; i++ {
		var val = c.withdrawTxs.Front()
		var msg = &PotEvent{
			Chain:   c.config.Code,
			Content: NewBlockMessage(val.TXN),
		}

		if c.height-val.Height+1 >= c.config.ConfirmTimes {
			msg.Event = T_WITHDRAW_CONFIRM
			if !c.wallet.Seek(val.TXN) {
				msg.Event = T_WITHDRAW_FAIL
			}
		} else if c.height-val.Height == 0 {
			msg.Event = T_WITHDRAW
			c.withdrawTxs.PushBack(val)
		} else {
			msg.Event = T_WITHDRAW_UPDATE
			c.withdrawTxs.PushBack(val)
		}
		msg.ID = c.getEventID(c.height, val.Height, msg.Event, val.Index)
		log.Debug().Msgf("New Event: %s", mustMarshal(msg))
		c.onMessage(msg)
	}
}

func (c *Chain) add(addrs []string) (height int64) {
	c.Lock()
	defer c.Unlock()
	for i, _ := range addrs {
		addr := addrs[i]
		c.addrs[addr] = true
	}
	addAddr(c.config.Code, c.height, addrs)
	return c.height
}

func (c *Chain) getEventID(currentHeight int64, realHeight int64, event EventType, idx int64) string {
	return fmt.Sprintf("%d%d%04d%03d%02d", currentHeight, realHeight, idx, c.config.Idx, event)
}

func mustMarshal(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
