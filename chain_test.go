package chainpot

import (
	"encoding/json"
	"github.com/fadeAce/claws/types"
	"os"
	"os/signal"
	"syscall"
	"testing"
)

func TestNewChainpot(t *testing.T) {
	var cp = NewChainpot(&Config{
		CachePath: "./log",
		Coins: []types.Coins{
			{CoinType: "eth", Url: "ws://127.0.0.1:8546"},
		},
	})
	cp.Register("eth")
	cp.Add(0, []string{"0x78aE889cd04Cb9274C2600d68CCc5058F43dB63e"})
	cp.Start(func(idx int, event *PotEvent) {
		b, _ := json.Marshal(event)
		println(idx, string(b))
	})

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cp.Reset()
	WaitExit()
	println("Save data and exit.")
}
