package chainpot

import (
	"github.com/fadeAce/claws"
	"github.com/fadeAce/claws/types"
)

type Chainpot struct {
	chains    []*Chain
	conf      map[string]*chainOption
	OnMessage func(idx int, event *PotEvent)
}

func NewChainpot(conf *types.Claws) *Chainpot {
	var obj = &Chainpot{
		chains: make([]*Chain, 128),
		conf:   make(map[string]*chainOption),
	}
	claws.SetupGate(conf)
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
