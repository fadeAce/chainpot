package chainpot

import (
	"context"
	"github.com/fadeAce/chainpot/queue"
	"github.com/fadeAce/claws"
	"github.com/fadeAce/claws/types"
	"math/big"
	"sync"
	"sync/atomic"
)

type EventType int

const (
	// NORMAL STATE
	T_DEPOSIT EventType = iota
	T_WITHDRAW
	T_DEPOSIT_UPDATE
	T_WITHDRAW_UPDATE
	T_WITHDRAW_CONFIRM
	T_DEPOSIT_CONFIRM

	// ABNORMAL STATE
	T_WITHDRAW_FAIL
	T_ERROR
)

const (

	// NORMAL STATE
	T_DEPOSIT = iota
	T_WITHDRAW
	T_DEPOSIT_UPDATE
	T_WITHDRAW_UPDATE
	T_WITHDRAW_CONFIRM
	T_DEPOSIT_CONFIRM

	// ABNORMAL STATE
	T_WITHDRAW_FAIL
)

type ChainType string

type BlockMessage struct {
	Hash   string
	From   string
	To     string
	Fee    string
	Amount string
}

type Config struct {
}

type ChainFunc func(poe PotEvent)

type PotEvent struct {
	Chain ChainType
	// deposit or
	Typ     string
	Content interface{}
}

// NewChainpot gives a new chainpot entrance
func NewChainpot(config *Config) *Chainpot {
	return &Chainpot{nodes: make(map[string]string)}
}

// Add add a new chain in hot deployment with no intervention to current listen loop
func (cp *Chainpot) Register(chainType ChainType, height int64) error {
	if _, ok := cp.nodes[string(chainType)]; ok {
		return errors.New("exist " + string(chainType) + " kind! please do not add it again")
	}
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
