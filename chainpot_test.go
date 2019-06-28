package chainpot

import (
	"context"
	"github.com/fadeAce/claws"
	"github.com/fadeAce/claws/types"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// TEST MLGB
func TestChainpot_Ready(t *testing.T) {
	initMock()
	conf := &Config{
		CachePath: "./log",
		Coins: []*CoinConf{
			{Code: "mlgb", URL: "ws://localhost:8546", Idx: 2, ConfirmTimes: 3},
		},
	}
	for i, _ := range conf.Coins {
		item := conf.Coins[i]
		item.Code = strings.ToLower(item.Code)
	}

	var obj = &Chainpot{
		chains: make([]*Chain, 128),
		conf:   make(map[int]*CoinConf),
	}
	if path, err := filepath.Abs(conf.CachePath); err != nil {
		log.Fatal().Msgf(err.Error())
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
		"mlgb": &MaskBuilder{},
	})
	for _, cfg := range conf.Coins {
		obj.conf[cfg.Idx] = cfg
	}

	obj.Register(2)
	pushBack(9, BlockMessage{
		Hash:   "0xda29054d35d1af9d54e5e8aafce62fec11c716c8bef67508e2dc4ae5e3882ebb",
		From:   "0x78ae889cd04cb9274c2600d68ccc5058f43db63e",
		To:     "0x54a298ee9fccbf0ad8e55bc641d3086b81a48c41",
		Fee:    "0.000247385",
		Amount: "0.01",
	})
	obj.Add(2, []string{"0x78ae889cd04cb9274c2600d68ccc5058f43db63e"})
	obj.Start(func(idx int, event *PotEvent) {})

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	obj.Reset()
	println("Save data and exit.")
}

func TestNewChainpot(t *testing.T) {
	var cp = NewChainpot(&Config{
		CachePath: "./log",
		Coins: []*CoinConf{
			{Code: "ETH", URL: "ws://localhost:8546", Idx: 1, ConfirmTimes: 7},
		},
	})
	cp.Register(1)
	cp.Add(1, []string{"0x78aE889cd04Cb9274C2600d68CCc5058F43dB63e"})
	cp.Start(func(idx int, event *PotEvent) {})

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cp.Reset()
	println("Save data and exit.")
}

// 重复添加
func TestChainpot_Add(t *testing.T) {
	var cp = NewChainpot(&Config{
		CachePath: "./log",
		Coins: []*CoinConf{
			{Code: "ETH", URL: "ws://localhost:8546", Idx: 1, ConfirmTimes: 7},
		},
	})
	cp.Register(1)
	cp.Add(1, []string{"0x78aE889cd04Cb9274C2600d68CCc5058F43dB63e"})
	cp.Add(1, []string{"0x78aE889cd04Cb9274C2600d68CCc5058F43dB63e"})
	cp.Add(1, []string{"0xE08f0bccBCa8192620259aA402b29f7b862575D3"})
	cp.Start(func(idx int, event *PotEvent) {})
	select {}
}

// 重复注册
func TestChainpot_Register(t *testing.T) {
	var cp = NewChainpot(&Config{
		CachePath: "./log",
		Coins: []*CoinConf{
			{Code: "ETH", URL: "ws://localhost:8546", Idx: 1, ConfirmTimes: 7},
		},
	})
	cp.Register(1)
	err := cp.Register(1)
	if err != nil {
		log.Error().Msg(err.Error())
	}

	cp.Add(1, []string{"0x78aE889cd04Cb9274C2600d68CCc5058F43dB63e"})
	cp.Start(func(idx int, event *PotEvent) {})
	select {}
}

func TestChainpot_Reset(t *testing.T) {
	var cp = NewChainpot(&Config{
		CachePath: "./log",
		Coins: []*CoinConf{
			{Code: "ETH", URL: "ws://localhost:8546", Idx: 1, ConfirmTimes: 7},
		},
	})
	cp.Register(1)
	cp.Add(1, []string{"0x78aE889cd04Cb9274C2600d68CCc5058F43dB63e"})
	cp.Start(func(idx int, event *PotEvent) {})

	cp.Reset()
	select {}
}
