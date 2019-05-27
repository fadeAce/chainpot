package chainpot

import (
	"encoding/json"
	"github.com/fadeAce/claws"
	"github.com/fadeAce/claws/types"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"testing"
)

func TestNewChainpot(t *testing.T) {
	cfg, err := ioutil.ReadFile("./test/claws.yml")
	if cfg == nil || err != nil {
		panic("shut down with no configuration")
		return
	}
	var conf types.Claws
	err = yaml.Unmarshal(cfg, &conf)
	claws.SetupGate(&conf)
	wallet := claws.Builder.BuildWallet("eth")

	var opt = &ChainOption{
		ConfirmTimes: 7,
		Chain:        "eth",
	}
	var chainpot = NewChainpot(opt, wallet)
	chainpot.Add([]string{"0x78aE889cd04Cb9274C2600d68CCc5058F43dB63e"})
	chainpot.OnMessage = func(msg *PotEvent) {
		b, _ := json.Marshal(msg)
		println(string(b))
	}
	var forever = make(chan bool)
	<-forever
}
