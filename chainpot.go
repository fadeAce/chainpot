package chainpot

import (
	"context"
	"errors"
	"github.com/fadeAce/claws"
	"github.com/fadeAce/claws/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
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
	wg      *sync.WaitGroup
	runmode string
)

func init() {
	wg = &sync.WaitGroup{}
	runmode = os.Getenv("RUN_MODE")
	if runmode == "" {
		runmode = "debug"
	}
	// it's currently no need to break json format
	//log.Logger = log.Output(zerolog.ConsoleWriter{
	//	Out:        os.Stdout,
	//	TimeFormat: "2006/01/02 15:04:05",
	//})
	level := zerolog.DebugLevel
	if runmode == "release" {
		level = zerolog.WarnLevel
	}
	zerolog.SetGlobalLevel(level)
}

type Chainpot struct {
	chains    []*chain
	conf      map[int]*Coins
	onMessage MessageHandler
}

type MessageHandler func(idx int, event *PotEvent)

func NewChainpot(conf *ChainConf) *Chainpot {
	var obj = &Chainpot{
		chains: make([]*chain, 128),
		conf:   make(map[int]*Coins),
	}
	for i, _ := range conf.Coins {
		item := conf.Coins[i]
		obj.conf[item.Idx] = &item
	}

	if path, err := filepath.Abs(conf.CachePath); err != nil {
		log.Fatal().Msgf(err.Error())
	} else {
		initStorage(path)
	}

	clawsConf := &types.Claws{
		Ctx:     context.Background(),
		Version: conf.Version,
		Eth: &types.EthConf{
			Name: conf.Eth.Name,
			Url:  conf.Eth.Url,
		},
		Btc: &types.BtcConf{
			Name:     conf.Btc.Name,
			Url:      conf.Btc.Url,
			User:     conf.Btc.User,
			Password: conf.Btc.Password,
			Network:  conf.Btc.Network,
		},
		Coins: make([]types.Coins, 0),
	}
	for _, item := range conf.Coins {
		clawsConf.Coins = append(clawsConf.Coins, types.Coins{
			CoinType:     item.CoinType,
			Chain:        item.Chain,
			Symbol:       item.Symbol,
			ContractAddr: item.ContractAddr,
		})
	}
	claws.SetupGate(clawsConf, nil)

	return obj
}

func (c *Chainpot) Register(idx int) error {
	var opt, exist = c.conf[idx]
	if !exist {
		return errors.New("configure not exist")
	}

	if c.chains[opt.Idx] != nil {
		return errors.New("repeat register")
	}

	var wallet = claws.Builder.BuildWallet(opt.Symbol)
	var chain = newChain(opt, wallet)
	c.chains[opt.Idx] = chain
	chain.onMessage = func(msg *PotEvent) {
		c.onMessage(opt.Idx, msg)
	}

	return nil
}

func (c *Chainpot) Add(idx int, addrs []string) map[string]int64 {
	chain := c.chains[idx]
	if chain != nil {
		return c.chains[idx].add(addrs)
	}
	panic("try to add address at non exist chain")
	return nil
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
func (c *Chainpot) Ready(idx int) bool {
	var opt, exist = c.conf[idx]
	if !exist {
		return false
	}
	return c.chains[opt.Idx] != nil
}

// reset chain which matched with given []idx
// if []idx is empty reset all
func (c *Chainpot) Reset(idx ...int) {
	wg = &sync.WaitGroup{}
	if len(idx) == 0 {
		for i, _ := range c.chains {
			if c.chains[i] != nil {
				wg.Add(1)
				c.chains[i].cancel()
			}
		}
		wg.Wait()

		for i, _ := range c.chains {
			if c.chains[i] != nil {
				clearCacheConfig(c.chains[i].config.Symbol)
				c.chains[i] = nil
			}
		}
		return
	}

	for _, i := range idx {
		if c.chains[i] != nil {
			wg.Add(1)
			c.chains[i].cancel()
		}
	}
	wg.Wait()
	for _, i := range idx {
		if c.chains[i] != nil {
			clearCacheConfig(c.chains[i].config.Symbol)
			c.chains[i] = nil
		}
	}
}

// call the function when process exit.
func (c *Chainpot) Stop() {
	wg = &sync.WaitGroup{}
	for i, _ := range c.chains {
		if c.chains[i] != nil {
			wg.Add(1)
			c.chains[i].cancel()
		}
	}
	wg.Wait()
}
