package chainpot

import "github.com/fadeAce/claws/types"

type BlockMessage struct {
	Hash   string
	From   string
	To     string
	Fee    string
	Amount string
}

func NewBlockMessage(tx types.TXN) *BlockMessage {
	return &BlockMessage{
		Hash:   tx.HexStr(),
		From:   tx.FromStr(),
		To:     tx.ToStr(),
		Fee:    tx.FeeStr(),
		Amount: tx.AmountStr(),
	}
}

func (c *BlockMessage) HexStr() string {
	return c.Hash
}

func (c *BlockMessage) FromStr() string {
	return c.From
}

func (c *BlockMessage) ToStr() string {
	return c.To
}

func (c *BlockMessage) FeeStr() string {
	return c.Fee
}

func (c *BlockMessage) AmountStr() string {
	return c.Amount
}

func (c *BlockMessage) SetStr(hash string) {
	c.Hash = hash
}
