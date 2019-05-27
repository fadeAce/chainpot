package main

import (
	"encoding/json"
	"github.com/fadeAce/chainpot"
	"github.com/fadeAce/claws"
	"github.com/fadeAce/claws/types"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg, err := ioutil.ReadFile("./test/claws.yml")
	if cfg == nil || err != nil {
		panic("shut down with no configuration")
		return
	}
	var conf types.Claws
	err = yaml.Unmarshal(cfg, &conf)
	claws.SetupGate(&conf)
	wallet := claws.Builder.BuildWallet("eth")

	var opt = &chainpot.ChainOption{
		ConfirmTimes: 7,
		Chain:        "eth",
	}
	var cp = chainpot.NewChainpot(opt, wallet)
	cp.Add([]string{"0x78aE889cd04Cb9274C2600d68CCc5058F43dB63e"})
	cp.OnMessage = func(msg *chainpot.PotEvent) {
		b, _ := json.Marshal(msg)
		println(string(b))
	}

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown Server ...")
}
