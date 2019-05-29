package chainpot

import (
	"context"
	"errors"
	"github.com/fadeAce/claws"
	"github.com/fadeAce/claws/types"
	"path/filepath"
)

const (
	CHAIN_ETH = iota
	CHAIN_BTC
	CHAIN_ERC20
)

type Chainpot struct {
	chains    []*Chain
	conf      map[string]*chainOption
	logPath   string
	onMessage MessageHandler
}

type Config struct {
	LogPath string
	Coins   []struct {
		CoinType string `yaml:"type"`
		Url      string `yml:"url"`
		//Idx      string `yml:"idx"`
	}
}

type MessageHandler func(idx int, event *PotEvent)

func NewChainpot(conf *Config) *Chainpot {
	var obj = &Chainpot{
		chains: make([]*Chain, 128),
		conf:   make(map[string]*chainOption),
	}
	if path, err := filepath.Abs(conf.LogPath); err != nil {
		panic(err)
	} else {
		obj.logPath = path
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

func (c *Chainpot) Register(chainName string, endpoint int64) error {
	var opt, exist = c.conf[chainName]
	opt.LogPath = c.logPath
	if !exist {
		return errors.New("configure not exist")
	}

	if c.chains[opt.IDX] != nil {
		return errors.New("repeat register")
	}

	opt.Endpoint = endpoint
	var wallet = claws.Builder.BuildWallet(opt.Chain)
	var chain = newChain(opt, wallet)
	c.chains[opt.IDX] = chain
	chain.onMessage = func(msg *PotEvent) {
		c.onMessage(opt.IDX, msg)
	}

	return nil
}

func (c *Chainpot) Add(idx int, addrs []string) (height int64, err error) {
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

// // if chain matched idx has been registered return true otherwise return false todo
func (c *Chainpot) Ready(idx int) bool {
	return false
}

// reset chain which matched with given []idx
// if []idx is nil reset all
func (c *Chainpot) Reset(idx ...int) {
	if len(idx) == 0 {
		// reset all todo
	}
	/* todo reset each one
	for _, v := range idx {

	}
	*/
}
