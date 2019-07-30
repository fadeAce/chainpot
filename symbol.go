package chainpot

import (
	"context"
	"github.com/fadeAce/chainpot/poterr"
	"github.com/fadeAce/claws"
	"github.com/fadeAce/claws/types"
	"github.com/rs/zerolog/log"
	"math/big"
	"strings"
	"sync"
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
	Symbol   string
	Chain    string
	CoinType string
	Event    EventType
	ID       int64
	Content  *BlockMessage
}

type contract struct {
	*Coins
	wallet claws.Wallet
}

// main structure for implement a set functions of a chain
type chain struct {
	*sync.Mutex
	origin         *contract
	contracts      []*contract
	addrs          map[string]int64
	eventID        int64
	syncedEndPoint bool
	depositTxs     *Queue
	withdrawTxs    *Queue
	storage        *storage
	noticer        chan *big.Int
	onMessage      func(msg *PotEvent)
	messageQueue   chan *PotEvent
	ctx            context.Context
	cancel         context.CancelFunc
	height         int64
	confirmTimes   int64
	endpoint       int64
}

// pot event iterator
func (c *PotEvent) Next(e EventType) *PotEvent {
	return &PotEvent{
		Symbol:  c.Symbol,
		Event:   e,
		ID:      c.ID + 1,
		Content: c.Content,
	}
}

type chain_option struct {
	ChainName    string
	Contracts    []*Coins
	ConfirmTimes int64
	Endpoint     int64
}

func newChain(opt *chain_option) *chain {
	ctx, cancel := context.WithCancel(context.Background())
	stg := newStorage(opt.ChainName)
	cache, addrs := getCacheConfig(opt.ChainName)

	chain := &chain{
		Mutex:        &sync.Mutex{},
		contracts:    make([]*contract, 0),
		addrs:        addrs,
		height:       opt.Endpoint,
		confirmTimes: opt.ConfirmTimes,
		endpoint:     cache.EndPoint,
		eventID:      cache.EventID,
		// todo: 128 is not quiet sufficient for concurrent consideration , need a much more flexible capacity
		messageQueue: make(chan *PotEvent, 128),
		depositTxs:   NewQueue(),
		withdrawTxs:  NewQueue(),
		storage:      stg,
		noticer:      make(chan *big.Int, 128),
		ctx:          ctx,
		cancel:       cancel,
	}

	for _, item := range opt.Contracts {
		if item.Chain == opt.ChainName {
			var obj = &contract{
				wallet: claws.Builder.BuildWallet(item.Symbol),
				Coins:  item,
			}
			if item.CoinType == "origin" {
				chain.origin = obj
			} else {
				chain.contracts = append(chain.contracts, obj)
			}
		}
	}

	return chain
}

func (c *chain) start() {
	log.Info().Msgf("%s start", strings.ToUpper(c.origin.Chain))

	go func() {
		err := c.origin.wallet.NotifyHead(c.ctx, func(num *big.Int) {
			var height = num.Int64()
			if c.origin.Chain == "eth" {
				height--
			}
			if height > c.height {
				c.height = height
				c.noticer <- big.NewInt(height)
				log.Info().Msgf("%d received new block from claws ", height)
				saveCacheConfig(c.origin.Chain, &cacheConfig{EndPoint: height, EventID: c.eventID}, nil)
			}
		})
		if err != nil {
			log.Error().Msgf("fatal error when starting head syncing: %s", err.Error())
		}
	}()

	go func() {
		for {
			select {
			case <-c.ctx.Done():
				saveCacheConfig(c.origin.Chain, &cacheConfig{EndPoint: c.height}, c.addrs)
				wg.Done()
				log.Info().Msgf("%s stopped, endpoint: %d", strings.ToUpper(c.origin.Chain), c.height)
				return
			case num := <-c.noticer:
				height := num.Int64()
				c.syncEndpoint(c.origin, height)
				c.syncBlock(c.origin, num, false)
				for _, item := range c.contracts {
					c.syncEndpoint(item, height)
					c.syncBlock(item, num, false)
				}
				c.emitter()
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
func (c *chain) syncBlock(cont *contract, num *big.Int, isOldBlock bool) {
	var height = num.Int64()
	// todo: shouldn't depend on syncing noticer but height calculation | or maybe suitable because of suitability
	//if cont.CoinType == "origin" {
	//	log.Info().Msgf("%s Synchronizing Block: %d", strings.ToUpper(c.origin.Chain), height)
	//}

	txns, err := cont.wallet.UnfoldTxs(context.Background(), num)
	if err != nil {
		return
	}

	for i, _ := range txns {
		var tx = txns[i]
		if cont.CoinType == "origin" && c.isContractTx(tx) {
			continue
		}

		var _, f1 = c.addrs[tx.FromStr()]
		var _, f2 = c.addrs[tx.ToStr()]
		if !f1 && !f2 {
			continue
		}

		var node = &Value{TXN: tx, Height: height, Index: int64(i), IsOldBlock: isOldBlock, EventID: c.eventID, Contract: cont}
		if tx.FromStr() == tx.ToStr() {
			c.messageQueue <- &PotEvent{
				Symbol: c.origin.Symbol,
				Event:  T_ERROR,
			}
		} else if f1 && f2 {
			c.withdrawTxs.Pend(node)
			c.eventID += c.confirmTimes
			var cp = *node

			c.depositTxs.Pend(&cp)
			c.eventID += c.confirmTimes
		} else if f1 {
			c.withdrawTxs.Pend(node)
			c.eventID += c.confirmTimes
		} else if f2 {
			c.depositTxs.Pend(node)
			c.eventID += c.confirmTimes
		}
	}
}

func (c *chain) syncEndpoint(cont *contract, currentHeight int64) {
	if c.eventID == 0 {
		c.eventID++
	}
	if c.syncedEndPoint || c.endpoint <= 0 {
		return
	}

	for i := c.endpoint - c.confirmTimes; i < currentHeight; i++ {
		c.syncBlock(cont, big.NewInt(i), true)
	}
	c.syncedEndPoint = true
}

// emit events
func (c *chain) emitter() {
	c.depositTxs.PopEach(func(i int, val *Value) {
		var event = &PotEvent{
			Chain:    val.Contract.Chain,
			CoinType: val.Contract.CoinType,
			Symbol:   val.Contract.Symbol,
			Content:  NewBlockMessage(val.TXN),
			ID:       val.EventID,
		}

		// todo: if coming block is not a new block, it need a check event is processing
		if val.IsOldBlock {
			event.Event = T_DEPOSIT
			c.messageQueue <- event
			for j := 0; j < int(c.confirmTimes-2); j++ {
				event = event.Next(T_DEPOSIT_UPDATE)
				c.messageQueue <- event
			}
			c.messageQueue <- event.Next(T_DEPOSIT_CONFIRM)
			return
		}

		if c.height-val.Height+1 >= c.confirmTimes {
			event = event.Next(T_DEPOSIT_CONFIRM)
			if !val.Contract.wallet.Seek(val.TXN) {
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
			Chain:    val.Contract.Chain,
			CoinType: val.Contract.CoinType,
			Symbol:   val.Contract.Symbol,
			Content:  NewBlockMessage(val.TXN),
			ID:       val.EventID,
		}

		if val.IsOldBlock {
			event.Event = T_WITHDRAW
			c.messageQueue <- event
			for j := 0; j < int(c.confirmTimes-2); j++ {
				event = event.Next(T_WITHDRAW_UPDATE)
				c.messageQueue <- event
			}
			c.messageQueue <- event.Next(T_WITHDRAW_CONFIRM)
			return
		}

		if c.height-val.Height+1 >= c.confirmTimes {
			event = event.Next(T_WITHDRAW_CONFIRM)
			if !val.Contract.wallet.Seek(val.TXN) {
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
	err := saveAddrs(c.origin.Chain, changed)
	if err != nil {
		log.Error().Str(poterr.AddErr.Error(), err.Error())
	}
	return records
}

func (c *chain) isContractTx(tx types.TXN) bool {
	var sig = false
	for _, item := range c.contracts {
		if item.ContractAddr == tx.ToStr() {
			sig = true
			break
		}
	}
	return sig
}
