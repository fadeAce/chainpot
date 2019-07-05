package chainpot

import (
	"context"
	"github.com/fadeAce/chainpot/poterr"
	"github.com/fadeAce/claws"
	"github.com/rs/zerolog/log"
	"math/big"
	"strings"
	"sync"
	"time"
)

type EventType int

// chain consistent indicating event types
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

// pot event carrier
type PotEvent struct {
	Chain   string
	Event   EventType
	ID      int64
	Content *BlockMessage
}

// main structure for implement a set functions of a chain
type chain struct {
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

// pot event iterator
func (c *PotEvent) Next(e EventType) *PotEvent {
	return &PotEvent{
		Chain:   c.Chain,
		Event:   e,
		ID:      c.ID + 1,
		Content: c.Content,
	}
}

func newChain(opt *CoinConf, wallet claws.Wallet) *chain {
	ctx, cancel := context.WithCancel(context.Background())
	stg := newStorage(opt.Code)
	cache, addrs := getCacheConfig(opt.Code)
	opt.Endpoint = cache.EndPoint
	chain := &chain{
		Mutex:     &sync.Mutex{},
		addrs:     addrs,
		height:    opt.Endpoint,
		syncedTxs: make(map[string]int64),
		config:    opt,
		wallet:    wallet,
		// todo: 128 is not quiet sufficient for concurrent consideration , need a much more flexible capacity
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

func (c *chain) start() {
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
				// todo: only been consumed it would cause a cache mark event

			}
		}
	}()
}

//
// @param isNextHeight bool "if current height is bigger than last"
func (c *chain) syncBlock(num *big.Int, isOldBlock bool, isNextHeight bool) {
	var height = num.Int64()
	// todo: shouldn't depend on syncing noticer but height calculation | or maybe suitable because of suitability
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
			c.withdrawTxs.Pend(node)
			c.eventID += c.config.ConfirmTimes
			var cp = *node

			c.depositTxs.Pend(&cp)
			c.eventID += c.config.ConfirmTimes
		} else if f1 {
			c.withdrawTxs.Pend(node)
			c.eventID += c.config.ConfirmTimes
		} else if f2 {
			c.depositTxs.Pend(node)
			c.eventID += c.config.ConfirmTimes
		}
	}

	if isNextHeight {
		c.emitter()
	}
}

func (c *chain) syncEndpoint(endpoint int64, currentHeight int64) {
	if c.eventID == 0 {
		c.eventID++
	}
	if c.syncedEndPoint || c.config.Endpoint <= 0 {
		return
	}

	// todo: reorg here may not be quiet suitable
	id, _ := c.storage.getEventID(endpoint - c.config.ConfirmTimes)
	c.eventID = id
	for i := endpoint - c.config.ConfirmTimes; i < currentHeight; i++ {
		c.syncBlock(big.NewInt(i), true, true)
	}
	c.syncedEndPoint = true
}

// emit events
func (c *chain) emitter() {
	c.depositTxs.PopEach(func(i int, val *Value) {
		var event = &PotEvent{
			Chain:   c.config.Code,
			Content: NewBlockMessage(val.TXN),
			ID:      val.EventID,
		}

		// todo: if coming block is not a new block, it need a check event is processing
		if val.IsOldBlock {
			event.Event = T_DEPOSIT
			c.messageQueue <- event
			for j := 0; j < int(c.config.ConfirmTimes-2); j++ {
				event = event.Next(T_DEPOSIT_UPDATE)
				c.messageQueue <- event
			}
			c.messageQueue <- event.Next(T_DEPOSIT_CONFIRM)
			return
		}

		if c.height-val.Height+1 >= c.config.ConfirmTimes {
			event = event.Next(T_DEPOSIT_CONFIRM)
			if !c.wallet.Seek(val.TXN) {
				return
			}
		} else if c.height-val.Height == 0 {
			event.Event = T_DEPOSIT
			c.depositTxs.Pend(val)
		} else {
			event = event.Next(T_DEPOSIT_UPDATE)
			val.EventID = event.ID
			c.depositTxs.Pend(val)
		}
		c.messageQueue <- event
	})

	c.withdrawTxs.PopEach(func(i int, val *Value) {
		var event = &PotEvent{
			Chain:   c.config.Code,
			Content: NewBlockMessage(val.TXN),
			ID:      val.EventID,
		}

		if val.IsOldBlock {
			event.Event = T_WITHDRAW
			c.messageQueue <- event
			for j := 0; j < int(c.config.ConfirmTimes-2); j++ {
				event = event.Next(T_WITHDRAW_UPDATE)
				c.messageQueue <- event
			}
			c.messageQueue <- event.Next(T_WITHDRAW_CONFIRM)
			return
		}

		if c.height-val.Height+1 >= c.config.ConfirmTimes {
			event = event.Next(T_WITHDRAW_CONFIRM)
			if !c.wallet.Seek(val.TXN) {
				return
			}
		} else if c.height-val.Height == 0 {
			event.Event = T_WITHDRAW
			c.withdrawTxs.Pend(val)
		} else {
			event = event.Next(T_WITHDRAW_UPDATE)
			val.EventID = event.ID
			c.withdrawTxs.Pend(val)
		}
		c.messageQueue <- event
	})
}

// add address to listen on chain
func (c *chain) add(addrs []string) (records map[string]int64) {
	c.Lock()
	defer c.Unlock()

	changed := make(map[string]int64)

	records = make(map[string]int64)
	for _, addr := range addrs {
		if height, exist := c.addrs[addr]; exist {
			records[addr] = height
		} else {
			c.addrs[addr] = c.height
			records[addr] = c.height
			changed[addr] = c.height
		}
	}
	err := saveAddrs(c.config.Code, changed)
	if err != nil {
		log.Error().Str(poterr.AddErr.Error(), err.Error())
	}
	return records
}
