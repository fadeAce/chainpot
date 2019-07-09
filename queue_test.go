package chainpot

import (
	"encoding/json"
	"fmt"
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

func TestSerializeInterface(t *testing.T) {
	a := &BlockMessage{
		Hash:   "a",
		From:   "b",
		To:     "c",
		Fee:    "d",
		Amount: "e",
	}
	b := &BlockMessage{
		Hash:   "q",
		From:   "w",
		To:     "e",
		Fee:    "r",
		Amount: "t",
	}
	va := &Value{
		TXN:        a,
		Height:     1,
		Index:      2,
		EventID:    3,
		IsOldBlock: true,
	}
	vb := &Value{
		TXN:        b,
		Height:     1,
		Index:      2,
		EventID:    3,
		IsOldBlock: false,
	}
	sa, err := json.Marshal(va)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(sa))
	sb, err := json.Marshal(vb)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(sb))

	// it's time for cover back
	var va2 = new(Value)
	va2.TXN = &BlockMessage{}
	err = json.Unmarshal(sa, va2)
	var vb2 = new(Value)
	err = json.Unmarshal(sb, vb2)


	a2 := va2.TXN.(*BlockMessage)
	fmt.Println(a2)
	b2 := vb2.TXN.(*BlockMessage)
	fmt.Println(b2)

	fmt.Println(va2.TXN.HexStr())
	fmt.Println(vb2.TXN.HexStr())
}
