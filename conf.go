package chainpot

import "context"

type ChainConf struct {
	CachePath string
	Ctx       context.Context
	Version   string   `yaml:"version"`
	Coins     []Coins  `yaml:"coins"`
	Eth       *EthConf `yaml:"chain_ethereum"`
	Btc       *BtcConf `yaml:"chain_bitcoin"`
}

type Coins struct {
	Idx          int
	Endpoint     int64
	ConfirmTimes int64
	CoinType     string `yaml:"type"`
	Chain        string `yaml:"chain"`
	ContractAddr string `yaml:"contract_addr"`
	Symbol       string `yaml:"symbol"`
}

type EthConf struct {
	Name string `yaml:"name"`
	Url  string `yaml:"url"`
}

type BtcConf struct {
	Name     string `yaml:"name"`
	Url      string `yaml:"url"`
	User     string `yml:"user"`
	Password string `yml:"password"`
	Network  string `yml:"network"`
}
