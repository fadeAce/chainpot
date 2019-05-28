package chainpot

import "github.com/fadeAce/claws/types"

type Value struct {
	TXN    types.TXN
	Height int64
	Index  int64
}

type Queue struct {
	tail  *Queue
	prev  *Queue
	next  *Queue
	len   int
	value *Value
}

func NewQueue() *Queue {
	var obj = &Queue{}
	return obj
}

func (c *Queue) Len() int {
	return c.len
}

func (c *Queue) PushBack(v *Value) {
	if c.value != nil {
		var node = &Queue{
			prev:  c.tail,
			value: v,
			len:   c.len,
		}
		node.len = c.len
		node.tail = node
		c.tail.next = node
		c.tail = node
	} else {
		c.value = v
		c.tail = c
	}
	c.len++
}

func (c *Queue) Front() (val *Value) {
	if c.next != nil {
		val = c.value
		var newFront = c.next
		c.value = newFront.value
		c.next = newFront.next
		c.prev = nil
	} else {
		val = c.value
		c.value = nil
		c.tail = nil
	}
	if c.len > 0 {
		c.len--
	}
	return
}
