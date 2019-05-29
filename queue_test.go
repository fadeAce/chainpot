package chainpot

import (
	"testing"
)

func TestNewQueue(t *testing.T) {
	var q = NewQueue()
	q.PushBack(&Value{Index: 9})
	q.PushBack(&Value{Index: 14})
	q.PushBack(&Value{Index: 10})
	q.PushBack(&Value{Index: 3})

	for q.Len() > 0 {
		var cur = q.Front()
		println(cur.Index)
	}
}

func BenchmarkQueue_PushBack(b *testing.B) {
	var Q = NewQueue()
	for i := 0; i < b.N; i++ {
		Q.PushBack(&Value{})
	}
}

func BenchmarkQueue_Front(b *testing.B) {
	var Q = NewQueue()
	for i := 0; i < b.N; i++ {
		Q.PushBack(&Value{})
	}

	for i := 0; i < b.N; i++ {
		Q.Front()
	}
}
