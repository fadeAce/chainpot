package chainpot

import (
	"fmt"
	"testing"
)

func TestNewTrie(t *testing.T) {
	s1 := "testing"
	s2 := "tester"
	s3 := "look"
	s4 := "looks"

	tri := NewPot()
	tri.Insert(s1)
	tri.Insert(s2)
	tri.Insert(s3)

	fmt.Println(tri.Search(s3))
	fmt.Println(tri.Search(s4))
}
