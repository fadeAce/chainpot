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

type PubLicChain uint8

const (
	Bitcoin PubLicChain = iota
	Ethereum
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

	level := zerolog.DebugLevel
	if runmode == "release" {
		level = zerolog.WarnLevel
	}
	zerolog.SetGlobalLevel(level)
}

type Chainpot struct {
	chains    []*chain
	conf      *ChainConf
	onMessage MessageHandler
}

type MessageHandler func(chain PubLicChain, event *PotEvent)

func NewChainpot(conf *ChainConf) *Chainpot {
	var obj = &Chainpot{
		chains: make([]*chain, 128),
		conf:   conf,
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

func (c *Chainpot) Register(chain PubLicChain) error {
	var chainName string
	var confirmTimes int64
	var contracts = make([]*Coins, 0)

	if chain == Ethereum {
		confirmTimes = c.conf.Eth.ConfirmTimes
		chainName = "eth"
	} else if chain == Bitcoin {
		confirmTimes = c.conf.Btc.ConfirmTimes
		chainName = "btc"
	}
	for i, _ := range c.conf.Coins {
		contracts = append(contracts, &c.conf.Coins[i])
	}

	var idx = int(chain)
	if c.chains[idx] != nil {
		return errors.New("repeat register")
	}

	var obj = newChain(&chain_option{
		ChainName:    chainName,
		ConfirmTimes: confirmTimes,
		Contracts:    contracts,
	})

	c.chains[idx] = obj
	obj.onMessage = func(msg *PotEvent) {
		c.onMessage(chain, msg)
	}

	return nil
}

func (c *Chainpot) Add(chain PubLicChain, addrs []string) map[string]int64 {
	var idx = int(chain)
	obj := c.chains[idx]
	if obj != nil {
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
func (c *Chainpot) Ready(chain PubLicChain) bool {
	var idx = int(chain)
	return c.chains[idx] != nil
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
				clearCacheConfig(c.chains[i].origin.Chain)
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
			clearCacheConfig(c.chains[i].origin.Symbol)
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
