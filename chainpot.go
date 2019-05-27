package chainpot

import (
	"context"
	"github.com/fadeAce/claws"
	"github.com/fadeAce/claws/types"
)

type Chainpot struct {
	chains    []*Chain
	conf      map[string]*chainOption
	OnMessage func(idx int, event *PotEvent)
}

type Config struct {
	Coins []struct {
		CoinType string `yaml:"type"`
		// RPC location is configured to wallet builder
		// like 127.0.0.1:8545
		Url string `yml:"url"`
	}
}

func NewChainpot(conf *Config) *Chainpot {
	var obj = &Chainpot{
		chains: make([]*Chain, 128),
		conf:   make(map[string]*chainOption),
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
	}
}

func (c *Chainpot) Add(idx int, addrs []string) (height int64) {
	return c.chains[idx].add(addrs)
}
