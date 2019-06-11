package chainpot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fadeAce/claws"
	"github.com/rs/zerolog/log"
	"math/big"
	"os"
	"strconv"
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
	ID      int64
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

type chainOption struct {
	ConfirmTimes int64
	Chain        string
	IDX          int
	Endpoint     int64
}

func newChain(opt *CoinConf, wallet claws.Wallet) *Chain {
	ctx, cancel := context.WithCancel(context.Background())
	chain := &Chain{
		Mutex:       &sync.Mutex{},
		addrs:       make(map[string]bool),
		syncedTxs:   make(map[string]int64),
		config:      opt,
		wallet:      wallet,
		depositTxs:  NewQueue(),
		withdrawTxs: NewQueue(),
		storage:     newStorage(opt.Code),
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
	go func() {
		c.wallet.NotifyHead(c.ctx, func(num *big.Int) {
			c.noticer <- num
		})
	}()

	go func() {
		var ticker = time.NewTicker(180 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-c.ctx.Done():
				saveCacheConfig(c.config.Code, &cacheConfig{EndPoint: c.height})
				wg.Done()
				println(fmt.Sprintf("Exit %s, endpoint: %d", c.config.Code, c.height))
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
					saveCacheConfig(c.config.Code, &cacheConfig{EndPoint: c.height})
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
		var _, f1 = c.addrs[tx.ToStr()]
		var _, f2 = c.addrs[tx.FromStr()]
		if !f1 && !f2 {
			continue
		}
		if _, exist := c.syncedTxs[tx.HexStr()]; exist {
			continue
		}

		block = append(block, NewBlockMessage(tx))
		c.syncedTxs[tx.HexStr()] = time.Now().UnixNano() / 1000000
		var node = &Value{TXN: tx, Height: height, Index: int64(i)}
		if (f1 || f2) && tx.FromStr() == tx.ToStr() {
			c.onMessage(&PotEvent{
				Chain: c.config.Code,
				ID:    c.getEventID(height, T_ERROR, int64(i)),
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
		msg.ID = c.getEventID(val.Height, msg.Event, val.Index)
		log.Info().Msgf("New Event: %s",mustMarshal(msg))
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
		msg.ID = c.getEventID(val.Height, msg.Event, val.Index)
		log.Info().Msgf("New Event: %s",mustMarshal(msg))
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

func (c *Chain) getEventID(height int64, event EventType, idx int64) int64 {
	var str = fmt.Sprintf("%d%011d%05d", c.config.Idx, height, idx)
	num, _ := strconv.Atoi(str)
	return int64(num)
}

func mustMarshal(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
