package lox

import (
	"fmt"
	"strconv"
	"time"
	"unicode/utf8"
)

func parameterMessage(fn string, expected, real int) string {
	return fn + ": expected " + strconv.Itoa(expected) + " arguments but got " + strconv.Itoa(real)
}

/////////////////// build in function

// -------- clock ----------------------------------------------------------------------
func NewClock() LoxCallable {
	return &Clock{}
}

type Clock struct{}

func (this *Clock) Arity() int {
	return 0
}

func (this *Clock) Call(interpreter *Interpreter, arguments []interface{}) (interface{}, error) {
	return float64(time.Now().UnixNano()), nil
}

func (this Clock) String() string {
	return "<native fn>"
}

// -------- len -------------------------------------------------------------------------
func NewLen() LoxCallable {
	return &Len{}
}

type Len struct{}

func (this *Len) Arity() int {
	return 1
}

func (this *Len) Call(interpreter *Interpreter, arguments []interface{}) (interface{}, error) {
	arg := arguments[0]
	switch v := arg.(type) {
	case string:
		return float64(utf8.RuneCountInString(v)), nil
	case LoxIterator:
		return float64(v.Len()), nil
	}
	return float64(1), nil
}

func (this Len) String() string {
	return "<native fn>"
}

// -------- string ----------------------------------------------------------------------
func NewString() LoxCallable {
	return &String{}
}

type String struct{}

func (this *String) Arity() int {
	return 1
}

func (this *String) Call(interpreter *Interpreter, arguments []interface{}) (interface{}, error) {
	arg := arguments[0]
	switch v := arg.(type) {
	case string:
		return v, nil
	default:
		return fmt.Sprintf("%v", arg), nil
	}
}

func (this String) String() string {
	return "<native fn>"
}
