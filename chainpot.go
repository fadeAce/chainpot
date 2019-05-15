package main

const (
	ETHEREUM ChainType = "eth"
	BITCOIN  ChainType = "btc"
	ERC20    ChainType = "erc20"
)

type ChainType string

type Chainpot struct {
}

type ChainFunc func(poe PotEvent)

type PotEvent struct {
	Chain ChainType
}

func NewChainpot() *Chainpot {
	return &Chainpot{}
}

// Add add a new chain in hot deployment with no intervention to current listen loop
// input all addresses in slice to be listened also with height
// return error if there exist
func (cp *Chainpot) Register(chainType ChainType, init []string, height int64) error {
	return nil
}

func (cp *Chainpot) Add(chainType ChainType, init string) (int64, error) {
	return 0, nil
}

func (cp *Chainpot) Subscribe(chainType ChainType, function ChainFunc) {

}
