package chainpot

import (
	"testing"
)

func TestNewQueue(t *testing.T) {
	var q = NewQueue()
	q.Pend(&Value{Index: 9})
	q.Pend(&Value{Index: 14})
	q.Pend(&Value{Index: 10})
	q.Pend(&Value{Index: 3})

	for q.Len() > 0 {
		var cur = q.Pop()
		println(cur.Index)
	}
}

func BenchmarkQueue_Pend(b *testing.B) {
	var Q = NewQueue()
	for i := 0; i < b.N; i++ {
		Q.Pend(&Value{})
	}
}

func BenchmarkQueue_Pop(b *testing.B) {
	var Q = NewQueue()
	for i := 0; i < b.N; i++ {
		Q.Pend(&Value{})
	}

	for i := 0; i < b.N; i++ {
		Q.Pop()
	}
}

func TestNewQueueNil(t *testing.T) {
	var q = NewQueue()
	q.Pop()
	q.Pend(&Value{Index: 9})
	q.Pend(&Value{Index: 14})
	q.Pend(&Value{Index: 10})
	q.Pend(&Value{Index: 3})

	for q.Len() > 0 {
		var cur = q.Pop()
		println(cur.Index)
	}
}
