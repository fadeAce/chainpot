package chainpot

import (
	"errors"
	"sync"
)

const (
	ETHEREUM ChainType = "eth"
	BITCOIN  ChainType = "btc"
	ERC20    ChainType = "erc20"
)

const (
	T_DEPOSIT = iota
	T_WITHDRAW
	T_DEPOSIT_UPDATE
	T_WITHDRAW_UPDATE
	T_WITHDRAW_CONFIRM
	T_DEPOSIT_CONFIRM
)

type ChainType string

type Chainpot struct {
	mux   sync.RWMutex
	nodes map[string]string
}

type ChainFunc func(poe PotEvent)

type PotEvent struct {
	Chain ChainType
	// deposit or
	Typ     string
	Content interface{}
}

// NewChainpot gives a new chainpot entrance
func NewChainpot() *Chainpot {
	return &Chainpot{nodes: make(map[string]string)}
}

// Add add a new chain in hot deployment with no intervention to current listen loop
func (cp *Chainpot) Register(chainType ChainType, rpcUrl string, height int64) error {
	if _, ok := cp.nodes[string(chainType)]; ok {
		return errors.New("exist " + string(chainType) + " kind! please do not add it again")
	}
	cp.mux.Lock()
	cp.nodes[string(chainType)] = rpcUrl
	cp.mux.Unlock()
	return nil
}

// Add register typed address for certain chain it could be invoked by anytime
func (cp *Chainpot) Add(chainType ChainType, init []string) (int64, error) {
	return 0, nil
}

// Subscribe never return
// when received new event it's caught by ChainFunc
func (cp *Chainpot) Subscribe(chainType ChainType, function ChainFunc) error {
	return nil
}

func (cp *Chainpot) Start() error {
	return nil
}
