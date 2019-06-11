package chainpot

import (
	"context"
	"github.com/fadeAce/claws"
	"github.com/fadeAce/claws/types"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
)

func newOfflineTestChainpot(conf *Config) *Chainpot {

	var obj = &Chainpot{
		chains: make([]*Chain, 128),
		conf:   make(map[int]*CoinConf),
	}
	if path, err := filepath.Abs(conf.CachePath); err != nil {
		panic(err)
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
	}, map[string]claws.WalletBuilder{
		"MLGB": &MaskBuilder{},
	})

	for _, cfg := range conf.Coins {
		obj.conf[cfg.Idx] = cfg
	}
	return obj

}

func TestNewChainpot(t *testing.T) {
	var cp = NewChainpot(&Config{
		CachePath: "./log",
		Coins: []*CoinConf{
			{
				Code:         "ETH",
				URL:          "ws://localhost:8546",
				Idx:          1,
				ConfirmTimes: 7,
			},
		},
	})
	cp.Register(1)
	cp.Add(1, []string{"0x78aE889cd04Cb9274C2600d68CCc5058F43dB63e"})
	cp.Start(func(idx int, event *PotEvent) {
	})

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cp.Reset()
	println("Save data and exit.")
}
