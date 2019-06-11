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
	"strings"
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

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "2006/01/02 15:04:05",
	})
	level := zerolog.DebugLevel
	if runmode == "release" {
		level = zerolog.WarnLevel
	}
	zerolog.SetGlobalLevel(level)
}

type Chainpot struct {
	chains    []*Chain
	conf      map[int]*CoinConf
	onMessage MessageHandler
}

type CoinConf struct {
	Code         string
	URL          string
	Idx          int
	ConfirmTimes int64
	Endpoint     int64
}

type Config struct {
	CachePath string
	Coins     []*CoinConf
}

type MessageHandler func(idx int, event *PotEvent)

func NewChainpot(conf *Config) *Chainpot {
	for i, _ := range conf.Coins {
		item := conf.Coins[i]
		item.Code = strings.ToLower(item.Code)
	}

	var obj = &Chainpot{
		chains: make([]*Chain, 128),
		conf:   make(map[int]*CoinConf),
	}
	if path, err := filepath.Abs(conf.CachePath); err != nil {
		log.Fatal().Msgf(err.Error())
	} else {
		initStorage(path)
	}

	coins := make([]types.Coins, 0)
	for _, item := range conf.Coins {
		coins = append(coins, types.Coins{Url: item.URL, CoinType: item.Code})
	}

	claws.SetupGate(&types.Claws{
		Ctx:     context.TODO(),
		Version: "0.0.1",
		Coins:   coins,
	}, nil)

	for _, cfg := range conf.Coins {
		obj.conf[cfg.Idx] = cfg
	}
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

	cache := getCacheConfig(opt.Code)
	opt.Endpoint = cache.EndPoint
	var wallet = claws.Builder.BuildWallet(opt.Code)
	var chain = newChain(opt, wallet)

	c.chains[opt.Idx] = chain
	chain.onMessage = func(msg *PotEvent) {
		c.onMessage(opt.Idx, msg)
	}

	return nil
}

func (c *Chainpot) Add(idx ChainType, addrs []string) (height int64, err error) {
	chain := c.chains[idx]
	if chain != nil {
		return c.chains[idx].add(addrs)
	}
	return 0, errors.New("idx not initialize")
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

//func (c *Chainpot) IDX(chainName string) (ChainType, error) {
//	var opt, exist = c.conf[chainName]
//	if !exist {
//		return 0, errors.New("configure not exist")
//	}
//	return ChainType(opt.IDX), nil
//}

// reset chain which matched with given []idx
// if []idx is empty reset all
func (c *Chainpot) Reset(idx ...int) {
	wg = &sync.WaitGroup{}
	if len(idx) == 0 {
		for i, _ := range c.chains {
			if c.chains[i] != nil {
				wg.Add(1)
				c.chains[i].cancel()
				clearCacheConfig(c.chains[i].config.Code)
				c.chains[i] = nil
			}
		}
	}
	for _, i := range idx {
		if c.chains[i] != nil {
			wg.Add(1)
			c.chains[i].cancel()
			clearCacheConfig(c.chains[i].config.Code)
			c.chains[i] = nil
		}
	}
	wg.Wait()
}
