package chainpot

import "github.com/fadeAce/claws/types"

const (
	HEAD = "_head"
)

type Value struct {
	TXN        types.TXN
	Height     int64
	Index      int64
	EventID    int64
	IsOldBlock bool
}

type Queue struct {
	data []*Value
}

func NewQueue() *Queue {
	var obj = &Queue{}
	return obj
}

func (q *Queue) Len() int {
	return len(q.data)
}

func (q *Queue) Pend(v *Value) {
	q.data = append(q.data, v)
}

// todo: it occurs panic when no value's left
func (q *Queue) Pop() *Value {
	var val = q.data[0]
	q.data = q.data[1:q.Len()]
	return val
}

// safe with wrapped pop
func (q *Queue) PopEach(f func(i int, v *Value)) {
	var m = q.Len()
	for i := 0; i < m; i++ {
		f(i, q.Pop())
	}
}

// type SafeQueue

type SafeQueue struct {
	data []*Value

	// storage based kv tx based safe queue

	// head is a string key represent head value
	head string
}

func (q *SafeQueue) Len() int {
	return len(q.data)
}

func (q *SafeQueue) Pend(v *Value) {
	q.data = append(q.data, v)
}

// todo: it occurs panic when no value's left
func (q *SafeQueue) Pop() *Value {
	var val = q.data[0]
	q.data = q.data[1:q.Len()]
	return val
}

// safe with wrapped pop
func (q *SafeQueue) PopEach(f func(i int, v *Value)) {
	var m = q.Len()
	for i := 0; i < m; i++ {
		f(i, q.Pop())
	}
}
