package main

import (
	"errors"
	"sync"
)

const (
	ETHEREUM ChainType = "eth"
	BITCOIN  ChainType = "btc"
	ERC20    ChainType = "erc20"
)

type ChainType string

type Chainpot struct {
	mux   sync.RWMutex
	nodes map[string]string
}

type ChainFunc func(poe PotEvent)

type PotEvent struct {
	Chain ChainType
}

func NewChainpot() *Chainpot {
	return &Chainpot{nodes: make(map[string]string)}
}

// Add add a new chain in hot deployment with no intervention to current listen loop
// input all addresses in slice to be listened also with height
// return error if there exist
// use rpc url to send rpc requests
func (cp *Chainpot) Register(chainType ChainType, rpcUrl string, init []string, height int64) error {
	if _, ok := cp.nodes[string(chainType)]; ok {
		return errors.New("exist " + string(chainType) + " kind! please do not add it again")
	}
	cp.mux.Lock()
	cp.nodes[string(chainType)] = rpcUrl
	cp.mux.Unlock()
	return nil
}

func (cp *Chainpot) Add(chainType ChainType, init string) (int64, error) {
	return 0, nil
}

func (cp *Chainpot) Subscribe(chainType ChainType, function ChainFunc) {

}

func (cp *Chainpot) Start() {
	cp.mux.RLock()
	for typ, _ := range cp.nodes {
		switch ChainType(typ) {
		case ETHEREUM:
			go func() {

			}()
		}
	}
	cp.mux.RUnlock()
}
