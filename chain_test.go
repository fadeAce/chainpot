package chainpot

import (
	"encoding/json"
	"github.com/fadeAce/claws/types"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"testing"
)

func TestNewChainpot(t *testing.T) {
	cfg, err := ioutil.ReadFile("./claws.yml")
	if cfg == nil || err != nil {
		panic("shut down with no configuration")
		return
	}
	var conf types.Claws
	err = yaml.Unmarshal(cfg, &conf)
	var cp = NewChainpot(&conf)
	cp.Register("eth", 0)
	cp.Add(0, []string{"0x78aE889cd04Cb9274C2600d68CCc5058F43dB63e"})
	cp.OnMessage = func(idx int, msg *PotEvent) {
		b, _ := json.Marshal(msg)
		println(idx, string(b))
	}

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown Server ...")
}
