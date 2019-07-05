package chainpot

import "github.com/fadeAce/claws/types"

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
