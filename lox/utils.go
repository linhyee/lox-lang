package lox

import (
	"fmt"
)

func IfFloat(expr bool, x, y float64) float64 {
	if expr {
		return x
	}
	return y
}

func FloatVal(v float64) string {
	text := fmt.Sprintf("%v", v)
	pos := len(text) - 2

	if pos > 0 && text[pos:] == ".0" {
		text = text[0:pos]
	}

	return text
}
