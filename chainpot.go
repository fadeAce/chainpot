package chainpot

import (
	"context"
	"errors"
	"fmt"
	"github.com/fadeAce/claws"
	"github.com/fadeAce/claws/types"
	"path/filepath"
	"sync"
)

type ChainType int

const (
	CHAIN_ETH ChainType = iota
	CHAIN_BTC
	CHAIN_ERC20
)

var (
	wg *sync.WaitGroup
)

func init() {
	wg = &sync.WaitGroup{}
}

type Chainpot struct {
	chains    []*Chain
	conf      map[string]*chainOption
	onMessage MessageHandler
}

type Config struct {
	CachePath string
	Coins     []types.Coins
}

type MessageHandler func(idx int, event *PotEvent)

func NewChainpot(conf *Config) *Chainpot {
	var obj = &Chainpot{
		chains: make([]*Chain, 128),
		conf:   make(map[string]*chainOption),
	}
	if path, err := filepath.Abs(conf.CachePath); err != nil {
		panic(err)
	} else {
		initStorage(path)
	}

	claws.SetupGate(&types.Claws{
		Ctx:     context.TODO(),
		Version: "0.0.1",
		Coins:   conf.Coins,
	})
	for _, cfg := range conf.Coins {
		obj.conf[cfg.CoinType] = &chainOption{
			Chain:        cfg.CoinType,
			ConfirmTimes: 7,
		}
	}
	return obj
}

func (c *Chainpot) Register(chainName string) error {
	var opt, exist = c.conf[chainName]
	if !exist {
		return errors.New("configure not exist")
	}

	if c.chains[opt.IDX] != nil {
		return errors.New("repeat register")
	}

	cache := getCacheConfig(chainName)
	opt.Endpoint = cache.EndPoint
	var wallet = claws.Builder.BuildWallet(opt.Chain)
	var chain = newChain(opt, wallet)

	c.chains[opt.IDX] = chain
	chain.onMessage = func(msg *PotEvent) {
		c.onMessage(opt.IDX, msg)
	}

	return nil
}

func (c *Chainpot) Add(idx ChainType, addrs []string) (height int64, err error) {
	return c.chains[idx].add(addrs)
}

func (c *Chainpot) Start(fn MessageHandler) {
	c.onMessage = fn
	for _, chain := range c.chains {
		if chain != nil {
			chain.start()
		}
	}
}

// // if chain matched idx has been registered return true otherwise return false
func (c *Chainpot) Ready(idx ChainType) bool {
	return c.chains[idx] != nil
}

func (c *Chainpot) IDX(chainName string) (ChainType, error) {
	var opt, exist = c.conf[chainName]
	if !exist {
		return 0, errors.New("configure not exist")
	}
	return ChainType(opt.IDX), nil
}

// reset chain which matched with given []idx
// if []idx is empty reset all
func (c *Chainpot) Reset(idx ...int) {
	if len(idx) == 0 {
		for i, _ := range c.chains {
			if c.chains[i] != nil {
				c.chains[i].stop()
			}
		}
	}
	for _, v := range idx {
		if c.chains[v] != nil {
			c.chains[v].stop()
		}
	}
}

func WaitExit() {
	wg.Wait()
}

func DisplayError(err error) {
	if err != nil {
		println(fmt.Sprintf("Error: %s", err.Error()))
	}
}
