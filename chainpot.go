package chainpot

import (
	"context"
	"github.com/fadeAce/claws"
	"github.com/fadeAce/claws/types"
	"path/filepath"
)

type Chainpot struct {
	storage   *storage
	chains    []*Chain
	conf      map[string]*chainOption
	OnMessage func(idx int, event *PotEvent)
}

type Config struct {
	LogPath string
	Coins   []struct {
		CoinType string `yaml:"type"`
		Url      string `yml:"url"`
		//Idx      string `yml:"idx"`
	}
}

func NewChainpot(conf *Config) *Chainpot {
	var obj = &Chainpot{
		chains: make([]*Chain, 128),
		conf:   make(map[string]*chainOption),
	}
	if path, err := filepath.Abs(conf.LogPath); err != nil {
		panic(err)
	} else {
		obj.storage = newStorage(path)
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

func (c *Chainpot) Register(chainName string, endpoint int64) {
	var opt, exist = c.conf[chainName]
	if !exist {
		return
	}

	opt.Endpoint = endpoint
	var wallet = claws.Builder.BuildWallet(opt.Chain)
	var chain = newChain(opt, wallet)
	c.chains[opt.IDX] = chain
	chain.onMessage = func(msg *PotEvent) {
		c.OnMessage(opt.IDX, msg)
		c.storage.append(msg)
	}
}

func (c *Chainpot) Add(idx int, addrs []string) (height int64) {
	return c.chains[idx].add(addrs)
}
