package stdlib

import (
	"fmt"
	gomath "math"
)

const (
	maxInt64 = int64(^uint64(0) >> 1)
	minInt64 = -maxInt64 - 1
)

func Math() Module {
	return Module{
		"E":      gomath.E,
		"MaxInt": maxInt64,
		"MinInt": minInt64,
		"PI":     gomath.Pi,
		"abs":    Function("math.abs", 1, mathAbs),
		"ceil":   Function("math.ceil", 1, mathUnary(gomath.Ceil)),
		"clamp":  Function("math.clamp", 3, mathClamp),
		"cos":    Function("math.cos", 1, mathUnary(gomath.Cos)),
		"exp":    Function("math.exp", 1, mathUnary(gomath.Exp)),
		"floor":  Function("math.floor", 1, mathUnary(gomath.Floor)),
		"isInf":  Function("math.isInf", 1, mathIsInf),
		"isNaN":  Function("math.isNaN", 1, mathIsNaN),
		"log":    Function("math.log", 1, mathUnary(gomath.Log)),
		"log10":  Function("math.log10", 1, mathUnary(gomath.Log10)),
		"max":    Function("math.max", -1, mathMax),
		"min":    Function("math.min", -1, mathMin),
		"mod":    Function("math.mod", 2, mathMod),
		"pow":    Function("math.pow", 2, mathPow),
		"round":  Function("math.round", 1, mathUnary(gomath.Round)),
		"sin":    Function("math.sin", 1, mathUnary(gomath.Sin)),
		"sqrt":   Function("math.sqrt", 1, mathUnary(gomath.Sqrt)),
		"tan":    Function("math.tan", 1, mathUnary(gomath.Tan)),
		"trunc":  Function("math.trunc", 1, mathUnary(gomath.Trunc)),
	}
}

func mathAbs(host Host, args []Value) (Value, error) {
	if value, ok := args[0].(int64); ok {
		if value == minInt64 {
			return gomath.Abs(float64(value)), nil
		}
		if value < 0 {
			return -value, nil
		}
		return value, nil
	}
	value, err := floatArg(host, args, 0, "value")
	if err != nil {
		return nil, err
	}
	return toIntIfExact(gomath.Abs(value)), nil
}

func mathClamp(host Host, args []Value) (Value, error) {
	value, err := floatArg(host, args, 0, "value")
	if err != nil {
		return nil, err
	}
	low, err := floatArg(host, args, 1, "low")
	if err != nil {
		return nil, err
	}
	high, err := floatArg(host, args, 2, "high")
	if err != nil {
		return nil, err
	}
	if low > high {
		return nil, fmt.Errorf("low must be less than or equal to high")
	}
	if value < low {
		value = low
	}
	if value > high {
		value = high
	}
	return toIntIfExact(value), nil
}

func mathIsInf(host Host, args []Value) (Value, error) {
	value, err := floatArg(host, args, 0, "value")
	if err != nil {
		return nil, err
	}
	return gomath.IsInf(value, 0), nil
}

func mathIsNaN(host Host, args []Value) (Value, error) {
	value, err := floatArg(host, args, 0, "value")
	if err != nil {
		return nil, err
	}
	return gomath.IsNaN(value), nil
}

func mathMax(host Host, args []Value) (Value, error) {
	return mathMinMax(host, args, false)
}

func mathMin(host Host, args []Value) (Value, error) {
	return mathMinMax(host, args, true)
}

func mathMod(host Host, args []Value) (Value, error) {
	leftInt, leftOK := args[0].(int64)
	rightInt, rightOK := args[1].(int64)
	if leftOK && rightOK {
		if rightInt == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return leftInt % rightInt, nil
	}
	left, err := floatArg(host, args, 0, "left")
	if err != nil {
		return nil, err
	}
	right, err := floatArg(host, args, 1, "right")
	if err != nil {
		return nil, err
	}
	if right == 0 {
		return nil, fmt.Errorf("division by zero")
	}
	return toIntIfExact(gomath.Mod(left, right)), nil
}

func mathPow(host Host, args []Value) (Value, error) {
	left, err := floatArg(host, args, 0, "base")
	if err != nil {
		return nil, err
	}
	right, err := floatArg(host, args, 1, "exponent")
	if err != nil {
		return nil, err
	}
	return toIntIfExact(gomath.Pow(left, right)), nil
}

func mathUnary(fn func(float64) float64) NativeFunc {
	return func(host Host, args []Value) (Value, error) {
		value, err := floatArg(host, args, 0, "value")
		if err != nil {
			return nil, err
		}
		return toIntIfExact(fn(value)), nil
	}
}

func mathMinMax(host Host, args []Value, wantMin bool) (Value, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("expected at least one number")
	}
	allInts := true
	best, err := floatArg(host, args, 0, "value")
	if err != nil {
		return nil, err
	}
	bestInt, intOK := args[0].(int64)
	allInts = allInts && intOK
	for i := 1; i < len(args); i++ {
		value, err := floatArg(host, args, i, "value")
		if err != nil {
			return nil, err
		}
		if intValue, ok := args[i].(int64); ok {
			if (wantMin && intValue < bestInt) || (!wantMin && intValue > bestInt) {
				bestInt = intValue
			}
		} else {
			allInts = false
		}
		if (wantMin && value < best) || (!wantMin && value > best) {
			best = value
		}
	}
	if allInts {
		return bestInt, nil
	}
	return toIntIfExact(best), nil
}
