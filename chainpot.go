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
	conf      map[int]*CoinConf
	onMessage MessageHandler
}

type CoinConf struct {
	// code used as coin type
	Code         string
	Idx          int
	ConfirmTimes int64
	Endpoint     int64
	// configuration for claws
	Chain           string
	Typ             string
	ContractAddress string
}

type Config struct {
	CachePath string
	Coins     []*CoinConf

	// chain configuration
	BtcConfig *BtcConfig

	EthConfig *EthConfig
}

type BtcConfig struct {
	Name     string
	Url      string
	User     string
	Password string
	Network  string
}

type EthConfig struct {
	Name string
	Url  string
}

type MessageHandler func(idx int, event *PotEvent)

func NewChainpot(conf *Config) *Chainpot {
	for i, _ := range conf.Coins {
		item := conf.Coins[i]
		item.Code = strings.ToLower(item.Code)
	}

	var obj = &Chainpot{
		chains: make([]*chain, 128),
		conf:   make(map[int]*CoinConf),
	}
	if path, err := filepath.Abs(conf.CachePath); err != nil {
		log.Fatal().Msgf(err.Error())
	} else {
		initStorage(path)
	}

	coins := make([]types.Coins, 0)
	for _, item := range conf.Coins {
		coins = append(coins, types.Coins{
			CoinType:     item.Typ,
			Chain:        item.Chain,
			ContractAddr: item.ContractAddress,
			Symbol:       item.Code,
		})
	}

	// setup btc chain
	var btcConf *types.BtcConf
	if conf.BtcConfig != nil {
		btcConf = &types.BtcConf{
			Name:     conf.BtcConfig.Name,
			Url:      conf.BtcConfig.Url,
			User:     conf.BtcConfig.User,
			Password: conf.BtcConfig.Password,
			Network:  conf.BtcConfig.Network,
		}
	}
	// setup eth chain
	var ethConf *types.EthConf
	if conf.EthConfig != nil {
		ethConf = &types.EthConf{
			Name: conf.EthConfig.Name,
			Url:  conf.EthConfig.Url,
		}
	}

	claws.SetupGate(&types.Claws{
		Ctx:     context.TODO(),
		Version: "0.0.1",
		Coins:   coins,
		Eth:     ethConf,
		Btc:     btcConf,
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

	var wallet = claws.Builder.BuildWallet(opt.Code)
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
				clearCacheConfig(c.chains[i].config.Code)
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
			clearCacheConfig(c.chains[i].config.Code)
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
