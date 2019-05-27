package queue

import "testing"

func TestNewQueue(t *testing.T) {
	var q = NewQueue()
	q.PushBack(&Value{Index: 9})
	q.PushBack(&Value{Index: 14})
	q.PushBack(&Value{Index: 10})

	for {
		if q.Len() == 0 {
			break
		}
		println(q.Front().Index)
	}
}
