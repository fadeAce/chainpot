package chainpot

import (
	"encoding/json"
	"testing"
)

func TestNewChainpot(t *testing.T) {
	var cp = NewChainpot(&Config{
		LogPath: "./log",
		Coins: []struct {
			CoinType string `yaml:"type"`
			Url      string `yml:"url"`
		}{
			{CoinType: "eth", Url: "ws://127.0.0.1:8546"},
		},
	})
	cp.Register("eth", 0)
	cp.Add(0, []string{"0x78aE889cd04Cb9274C2600d68CCc5058F43dB63e"})
	cp.OnMessage = func(idx int, msg *PotEvent) {
		b, _ := json.Marshal(msg)
		println(idx, string(b))
	}

	select {}
}
