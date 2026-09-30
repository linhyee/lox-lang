package stdlib

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Value = any

type NativeFunc func(host Host, args []Value) (Value, error)

type Native struct {
	Name  string
	Arity int
	Fn    NativeFunc
}

type Module map[string]Value

type Host interface {
	Args() []string
	Stdout() io.Writer
	Stderr() io.Writer
	Stringify(Value) string
	TypeName(Value) string
	IsNumber(Value) bool
	AsInt64(Value) (int64, bool)
	AsFloat64(Value) (float64, bool)
	NewList([]Value) Value
	ListItems(Value) ([]Value, bool)
	NewMap(map[string]Value) Value
	MapItems(Value) (map[string]Value, bool)
	ModuleExports(Value) (map[string]Value, bool)
}

func Function(name string, arity int, fn NativeFunc) *Native {
	return &Native{Name: name, Arity: arity, Fn: fn}
}

func Modules() map[string]Module {
	return map[string]Module{
		"std":    Std(),
		"os":     OS(),
		"string": String(),
		"json":   JSON(),
		"math":   Math(),
		"times":  Times(),
		"regexp": Regexp(),
	}
}

func stringArg(args []Value, index int, name string) (string, error) {
	value, ok := args[index].(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", name)
	}
	return value, nil
}

func intArg(host Host, args []Value, index int, name string) (int64, error) {
	value, ok := host.AsInt64(args[index])
	if !ok {
		return 0, fmt.Errorf("%s must be an integer", name)
	}
	return value, nil
}

func floatArg(host Host, args []Value, index int, name string) (float64, error) {
	value, ok := host.AsFloat64(args[index])
	if !ok {
		return 0, fmt.Errorf("%s must be a number", name)
	}
	return value, nil
}

func falsey(value Value) bool {
	if value == nil {
		return true
	}
	if b, ok := value.(bool); ok {
		return !b
	}
	return false
}

func parseNumber(text string) (Value, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("cannot parse empty string as number")
	}
	if strings.ContainsAny(text, ".eE") {
		value, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number %q", text)
		}
		return value, nil
	}
	value, err := strconv.ParseInt(text, 10, 64)
	if err == nil {
		return value, nil
	}
	floatValue, floatErr := strconv.ParseFloat(text, 64)
	if floatErr != nil {
		return nil, fmt.Errorf("invalid number %q", text)
	}
	return floatValue, nil
}

func toIntIfExact(value float64) Value {
	i := int64(value)
	if float64(i) == value {
		return i
	}
	return value
}

func listFromStrings(host Host, values []string) Value {
	items := make([]Value, len(values))
	for i, value := range values {
		items[i] = value
	}
	return host.NewList(items)
}
