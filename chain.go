package chainpot

import (
	"context"
	"encoding/json"
	"github.com/fadeAce/claws"
	"github.com/rs/zerolog/log"
	"math/big"
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

func (c *PotEvent) Next(e EventType) *PotEvent {
	return &PotEvent{
		Chain:   c.Chain,
		Event:   e,
		ID:      c.ID + 1,
		Content: c.Content,
	}
}

type Chain struct {
	*sync.Mutex
	addrs          map[string]int64
	syncedTxs      map[string]int64
	config         *CoinConf
	height         int64
	eventID        int64
	syncedEndPoint bool
	wallet         claws.Wallet
	depositTxs     *Queue
	withdrawTxs    *Queue
	storage        *storage
	noticer        chan *big.Int
	onMessage      func(msg *PotEvent)
	messageQueue   chan *PotEvent
	ctx            context.Context
	cancel         context.CancelFunc
}

func newChain(opt *CoinConf, wallet claws.Wallet) *Chain {
	ctx, cancel := context.WithCancel(context.Background())
	stg := newStorage(opt.Code)
	cache, addrs := getCacheConfig(opt.Code)
	opt.Endpoint = cache.EndPoint
	chain := &Chain{
		Mutex:        &sync.Mutex{},
		addrs:        addrs,
		height:       opt.Endpoint,
		syncedTxs:    make(map[string]int64),
		config:       opt,
		wallet:       wallet,
		messageQueue: make(chan *PotEvent, 128),
		depositTxs:   NewQueue(),
		withdrawTxs:  NewQueue(),
		storage:      stg,
		noticer:      make(chan *big.Int, 128),
		ctx:          ctx,
		cancel:       cancel,
	}
	return chain
}

func (c *Chain) start() {
	log.Info().Msgf("%s start", strings.ToUpper(c.config.Code))

	go func() {
		err := c.wallet.NotifyHead(c.ctx, func(num *big.Int) {
			c.noticer <- num
			//log.Info().Msgf("%s received new block from claws ", num.String())
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
				var isNextHeight = false
				if height > c.height {
					c.height = height
					isNextHeight = true
				}
				c.syncEndpoint(c.config.Endpoint, height)
				c.syncBlock(num, false, isNextHeight)
			case event := <-c.messageQueue:
				log.Debug().Msgf("New Event: %s", mustMarshal(event))
				c.onMessage(event)
			}
		}
	}()
}

// @param isNextHeight bool "if current height is bigger than last"
func (c *Chain) syncBlock(num *big.Int, isOldBlock bool, isNextHeight bool) {
	var height = num.Int64()
	if !isOldBlock {
		c.storage.setEventID(height, c.eventID)
	}

	log.Info().Msgf("%s Synchronizing Block: %d", strings.ToUpper(c.config.Code), height)
	txns, err := c.wallet.UnfoldTxs(context.Background(), num)
	if err != nil {
		return
	}

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

		c.syncedTxs[tx.HexStr()] = time.Now().UnixNano() / 1000000
		var node = &Value{TXN: tx, Height: height, Index: int64(i), IsOldBlock: isOldBlock, EventID: c.eventID}
		if tx.FromStr() == tx.ToStr() {
			c.messageQueue <- &PotEvent{
				Chain: c.config.Code,
				Event: T_ERROR,
			}
		} else if f1 && f2 {
			c.withdrawTxs.PushBack(node)
			c.eventID += c.config.ConfirmTimes
			var cp = *node
			c.depositTxs.PushBack(&cp)
			c.eventID += c.config.ConfirmTimes
		} else if f1 {
			c.withdrawTxs.PushBack(node)
			c.eventID += c.config.ConfirmTimes
		} else if f2 {
			c.depositTxs.PushBack(node)
			c.eventID += c.config.ConfirmTimes
		}
	}

	if isNextHeight {
		c.emitter()
	}
}

func (c *Chain) syncEndpoint(endpoint int64, currentHeight int64) {
	if c.eventID == 0 {
		c.eventID++
	}
	if c.syncedEndPoint || c.config.Endpoint <= 0 {
		return
	}

	id, _ := c.storage.getEventID(endpoint - c.config.ConfirmTimes)
	c.eventID = id
	for i := endpoint - c.config.ConfirmTimes; i < currentHeight; i++ {
		c.syncBlock(big.NewInt(i), true, true)
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
			ID:      val.EventID,
		}

		if val.IsOldBlock {
			msg.Event = T_DEPOSIT
			c.messageQueue <- msg
			for j := 0; j < int(c.config.ConfirmTimes-2); j++ {
				msg = msg.Next(T_DEPOSIT_UPDATE)
				c.messageQueue <- msg
			}
			c.messageQueue <- msg.Next(T_DEPOSIT_CONFIRM)
			continue
		}

		if c.height-val.Height+1 >= c.config.ConfirmTimes {
			msg = msg.Next(T_DEPOSIT_CONFIRM)
			if !c.wallet.Seek(val.TXN) {
				continue
			}
		} else if c.height-val.Height == 0 {
			msg.Event = T_DEPOSIT
			c.depositTxs.PushBack(val)
		} else {
			msg = msg.Next(T_DEPOSIT_UPDATE)
			val.EventID = msg.ID
			c.depositTxs.PushBack(val)
		}
		c.messageQueue <- msg
	}

	var n = c.withdrawTxs.Len()
	for i := 0; i < n; i++ {
		var val = c.withdrawTxs.Front()
		var msg = &PotEvent{
			Chain:   c.config.Code,
			Content: NewBlockMessage(val.TXN),
			ID:      val.EventID,
		}

		if val.IsOldBlock {
			msg.Event = T_WITHDRAW
			c.messageQueue <- msg
			for j := 0; j < int(c.config.ConfirmTimes-2); j++ {
				msg = msg.Next(T_WITHDRAW_UPDATE)
				c.messageQueue <- msg
			}
			c.messageQueue <- msg.Next(T_WITHDRAW_CONFIRM)
			continue
		}

		if c.height-val.Height+1 >= c.config.ConfirmTimes {
			msg = msg.Next(T_WITHDRAW_CONFIRM)
			if !c.wallet.Seek(val.TXN) {
				continue
			}
		} else if c.height-val.Height == 0 {
			msg.Event = T_WITHDRAW
			c.withdrawTxs.PushBack(val)
		} else {
			msg = msg.Next(T_WITHDRAW_UPDATE)
			val.EventID = msg.ID
			c.withdrawTxs.PushBack(val)
		}
		c.messageQueue <- msg
	}
}

func (c *Chain) add(addrs []string) (records map[string]int64) {
	c.Lock()
	defer c.Unlock()

	records = make(map[string]int64)
	for _, addr := range addrs {
		if height, exist := c.addrs[addr]; exist {
			records[addr] = height
		} else {
			c.addrs[addr] = c.height
			records[addr] = c.height
		}
	}
	saveAddrs(c.config.Code, records)
	return records
}

func mustMarshal(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func ToString(v interface{}) string {
	if num, ok := v.(int); ok {
		return strconv.Itoa(num)
	} else if num, ok := v.(int64); ok {
		return strconv.Itoa(int(num))
	}
	return ""
}
