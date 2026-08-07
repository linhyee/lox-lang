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
	case int64:
		switch bv := b.(type) {
		case int64:
			return av == bv
		case float64:
			return float64(av) == bv
		default:
			return false
		}
	case float64:
		switch bv := b.(type) {
		case int64:
			return av == float64(bv)
		case float64:
			return av == bv
		default:
			return false
		}
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
	case int64:
		return strconv.FormatInt(v, 10)
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

func IsNumber(value Value) bool {
	switch value.(type) {
	case int64, float64:
		return true
	default:
		return false
	}
}

func AsInt64(value Value) (int64, bool) {
	switch v := value.(type) {
	case int64:
		return v, true
	case float64:
		i := int64(v)
		return i, float64(i) == v
	default:
		return 0, false
	}
}

func AsFloat64(value Value) (float64, bool) {
	switch v := value.(type) {
	case int64:
		return float64(v), true
	case float64:
		return v, true
	default:
		return 0, false
	}
}

func numericBinary(left, right Value, intOp func(int64, int64) Value, floatOp func(float64, float64) Value) (Value, bool) {
	leftInt, leftIsInt := left.(int64)
	rightInt, rightIsInt := right.(int64)
	if leftIsInt && rightIsInt {
		return intOp(leftInt, rightInt), true
	}

	leftFloat, ok := AsFloat64(left)
	if !ok {
		return nil, false
	}
	rightFloat, ok := AsFloat64(right)
	if !ok {
		return nil, false
	}
	return floatOp(leftFloat, rightFloat), true
}
