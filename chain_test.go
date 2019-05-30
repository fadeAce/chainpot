package chainpot

import (
	"encoding/json"
	"github.com/boltdb/bolt"
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
	cp.Start(func(idx int, event *PotEvent) {
		b, _ := json.Marshal(event)
		println(idx, string(b))
	})
	forever := make(chan bool)
	<-forever
}

func TestChainpot_Add(t *testing.T) {
	var s = newStorage("/Users/caster/go/src/github.com/fadeAce/chainpot/log", "eth")
	db, _ := s.getDB(4460424)
	db.View(func(tx *bolt.Tx) error {
		tx.Bucket([]byte("eth")).ForEach(func(k, v []byte) error {
			println(string(k), string(v))
			return nil
		})
		return nil
	})
}
