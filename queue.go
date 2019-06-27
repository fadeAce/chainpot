package chainpot

import "github.com/fadeAce/claws/types"

type Value struct {
	TXN        types.TXN
	Height     int64
	Index      int64
	IsOldBlock bool
}

type Queue struct {
	data []*Value
}

func NewQueue() *Queue {
	var obj = &Queue{}
	return obj
}

func (c *Queue) Len() int {
	return len(c.data)
}

func (c *Queue) PushBack(v *Value) {
	c.data = append(c.data, v)
}

func (c *Queue) Front() *Value {
	var val = c.data[0]
	c.data = c.data[1:c.Len()]
	return val
}
