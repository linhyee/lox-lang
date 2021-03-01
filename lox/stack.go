package lox

import "fmt"

func NewStack() *Stack {
	return &Stack{top: -1, elem: make([]interface{}, 0)}
}

type Stack struct {
	top  int
	elem []interface{}
}

func (s *Stack) Get(index int) (interface{}, error) {
	if index < 0 || index > s.top {
		return nil, NewStackError(index, "access illegal address.")
	}
	return s.elem[index], nil
}

func (s *Stack) Size() int {
	return s.top + 1
}

func (s *Stack) IsEmpty() bool {
	return s.top < 0
}

func (s *Stack) Top() interface{} {
	return s.elem[s.top]
}

func (s *Stack) expand() {
	elem := make([]interface{}, s.top+1<<2)
	copy(elem, s.elem)
	s.elem = elem
}

func (s *Stack) Push(value interface{}) {
	if s.top++; s.top >= len(s.elem) {
		s.expand()
	}
	s.elem[s.top] = value
}

func (s *Stack) Pop() (interface{}, error) {
	if s.top < 0 {
		return nil, NewStackError(s.top, "stack empty.")
	}
	value := s.elem[s.top]
	s.elem[s.top] = nil
	s.top--
	return value, nil
}

func (s Stack) String() string {
	return fmt.Sprintf("stack info: <top,%d>, <size,%d>, <elems, %v >",
		s.top, s.Size(), s.elem)
}
