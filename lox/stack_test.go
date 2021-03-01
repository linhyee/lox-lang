package lox

import (
	"fmt"
	"testing"
)

func TestStack(t *testing.T) {
	s := NewStack()
	s.Push(false)
	s.Push("abc")
	s.Push(1)
	s.Push(1.234)
	s.Push(struct{}{})
	s.Push(true)
	fmt.Println(s)
	fmt.Println("top:", s.Top())
	fmt.Println("is empty ? ", s.IsEmpty())
	fmt.Println(s.Pop())
	fmt.Println(s.Pop())
	s.Pop()
	s.Pop()
	s.Pop()
	s.Pop()
	s.Pop()
	s.Pop()
	fmt.Println(s.Pop())
	fmt.Println(s.Pop())
	fmt.Println(s)
}
