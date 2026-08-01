package glox

import (
	"fmt"
	"math"
	"strconv"
)

type Value interface{}

func IsFalsey(value Value) bool {
	if value == nil {
		return true
	}
	if b, ok := value.(bool); ok {
		return !b
	}
	return false
}

func ValuesEqual(a, b Value) bool {
	switch av := a.(type) {
	case nil:
		return b == nil
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	case float64:
		bv, ok := b.(float64)
		return ok && av == bv
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	default:
		return a == b
	}
}

func Stringify(value Value) string {
	switch v := value.(type) {
	case nil:
		return "nil"
	case bool:
		if v {
			return "true"
		}
		return "false"
	case float64:
		if math.IsInf(v, 0) || math.IsNaN(v) {
			return fmt.Sprintf("%g", v)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}
