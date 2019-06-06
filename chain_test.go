package chainpot

import (
	"encoding/json"
	"os"
	"os/signal"
	"syscall"
	"testing"
)

func TestNewChainpot(t *testing.T) {
	var cp = NewChainpot(&Config{
		CachePath: "./log",
		Coins: []*CoinConf{
			//{CoinType: "eth", Url: "ws://127.0.0.1:8546"},
			{Code: "MLGB", URL: "ws://127.0.0.1:8546", Idx: 1, ConfirmTimes: 7},
		},
	})
	cp.Register(1)
	cp.Add(1, []string{"0x78aE889cd04Cb9274C2600d68CCc5058F43dB63e"})
	cp.Start(func(idx int, event *PotEvent) {
		b, _ := json.Marshal(event)
		println(idx, string(b))
	})

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cp.Reset()
	println("Save data and exit.")
}
