package queue

import "github.com/fadeAce/claws/types"

type Value struct {
	TXN    types.TXN
	Height int64
}
type Queue struct {
	data chan *Value
	len  int
}

func NewQueue(buf int) *Queue {
	var obj = &Queue{
		data: make(chan *Value, buf),
	}
	return obj
}

func (c *Queue) Len() int {
	return c.len
}

func (c *Queue) PushBack(v *Value) {
	c.len++
	c.data <- v
}

func (c *Queue) Front() *Value {
	ele := <-c.data
	c.len--
	return ele
}
