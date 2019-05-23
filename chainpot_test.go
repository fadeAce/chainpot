package chainpot

import (
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
	// first of all setup gate
	claws.SetupGate(&conf)
	wallet := claws.Builder.BuildWallet("eth")
	//b := wallet.NewAddr()
	//fmt.Println(b)

	//num := big.NewInt(4419795)
	//txns, err := wallet.UnfoldTxs(conf.Ctx, big.NewInt(4356126))
	//for _, v := range txns {
	//	fmt.Println("from ", v.FromStr(), " to ", v.ToStr(), " hash ", v.HexStr())
	//	fmt.Println(" fee ", v.FeeStr(), " amount ", v.AmountStr())
	//}

	var opt = &ChainOption{
		ConfirmTimes: 7,
		Code:"eth",
	}
	var chainpot = NewChainpot(opt, wallet)
	chainpot.OnMessage(func(msgs []BlockMessage) {
		
	})
}
