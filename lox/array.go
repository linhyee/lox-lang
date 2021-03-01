package lox

import (
	"bytes"
	"fmt"
)

func NewLoxArray(items []interface{}) LoxIterator {
	return &LoxArray{items: items, size: len(items)}
}

type LoxArray struct {
	size  int
	items []interface{}
}

func (this *LoxArray) Len() int {
	return this.size
}

func (this *LoxArray) Add(item interface{}) {
	this.items = append(this.items, item)
	this.size = len(this.items)
}

func (this *LoxArray) Get(index int) (interface{}, error) {
	if index < this.size && index >= 0 {
		return this.items[index], nil
	}
	return nil, NewIllegalIndexError(index, "access illegal index boundary.")
}

func (this *LoxArray) Set(index int, value interface{}) error {
	if index >= this.size || index < 0 {
		return NewIllegalIndexError(index, "set illegal index boundary")
	}
	this.items[index] = value
	return nil
}

func (this LoxArray) String() string {
	var out bytes.Buffer
	out.WriteString("[")
	for i, item := range this.items {
		var s string
		switch v := item.(type) {
		case float64:
			s = FloatVal(v)
		case string:
			s = v
		case nil:
			s = "nil"
		case bool:
			if v {
				s = "true"
			} else {
				s = "false"
			}
		default:
			if str, ok := v.(fmt.Stringer); ok {
				s = str.String()
			}
		}
		out.WriteString(s)
		if i != len(this.items)-1 {
			out.WriteString(",")
		}
	}
	out.WriteString("]")
	return out.String()
}
