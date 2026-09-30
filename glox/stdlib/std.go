package stdlib

import (
	"errors"
	"fmt"
	"sort"
)

func Std() Module {
	return Module{
		"assert":    Function("std.assert", -1, stdAssert),
		"bool":      Function("std.bool", 1, stdBool),
		"clone":     Function("std.clone", 1, stdClone),
		"eprint":    Function("std.eprint", -1, stdEprint),
		"eprintln":  Function("std.eprintln", -1, stdEprintln),
		"has":       Function("std.has", 2, stdHas),
		"isBoolean": Function("std.isBoolean", 1, typePredicate("boolean")),
		"isList":    Function("std.isList", 1, typePredicate("list")),
		"isMap":     Function("std.isMap", 1, typePredicate("map")),
		"isModule":  Function("std.isModule", 1, typePredicate("module")),
		"isNil":     Function("std.isNil", 1, typePredicate("nil")),
		"isNumber":  Function("std.isNumber", 1, typePredicate("number")),
		"isString":  Function("std.isString", 1, typePredicate("string")),
		"keys":      Function("std.keys", 1, stdKeys),
		"number":    Function("std.number", 1, stdNumber),
		"panic":     Function("std.panic", 1, stdPanic),
		"print":     Function("std.print", -1, stdPrint),
		"println":   Function("std.println", -1, stdPrintln),
		"range":     Function("std.range", -1, stdRange),
		"string":    Function("std.string", 1, stdString),
		"typeOf":    Function("std.typeOf", 1, stdTypeOf),
		"values":    Function("std.values", 1, stdValues),
	}
}

func stdAssert(host Host, args []Value) (Value, error) {
	if len(args) != 1 && len(args) != 2 {
		return nil, fmt.Errorf("std.assert expects 1 or 2 arguments but got %d", len(args))
	}
	if !falsey(args[0]) {
		return true, nil
	}
	message := "assertion failed"
	if len(args) == 2 {
		message = host.Stringify(args[1])
	}
	return nil, errors.New(message)
}

func stdBool(host Host, args []Value) (Value, error) {
	return !falsey(args[0]), nil
}

func stdClone(host Host, args []Value) (Value, error) {
	if items, ok := host.ListItems(args[0]); ok {
		return host.NewList(append([]Value(nil), items...)), nil
	}
	if items, ok := host.MapItems(args[0]); ok {
		clone := make(map[string]Value, len(items))
		for key, value := range items {
			clone[key] = value
		}
		return host.NewMap(clone), nil
	}
	return args[0], nil
}

func stdEprint(host Host, args []Value) (Value, error) {
	writeValues(host.Stderr(), host, args, false)
	return nil, nil
}

func stdEprintln(host Host, args []Value) (Value, error) {
	writeValues(host.Stderr(), host, args, true)
	return nil, nil
}

func stdHas(host Host, args []Value) (Value, error) {
	key, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("key must be a string")
	}
	if values, ok := host.MapItems(args[0]); ok {
		_, exists := values[key]
		return exists, nil
	}
	if values, ok := host.ModuleExports(args[0]); ok {
		_, exists := values[key]
		return exists, nil
	}
	return false, nil
}

func stdKeys(host Host, args []Value) (Value, error) {
	values, ok := keyedValues(host, args[0])
	if !ok {
		return nil, fmt.Errorf("value must be a map or module")
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	items := make([]Value, len(keys))
	for i, key := range keys {
		items[i] = key
	}
	return host.NewList(items), nil
}

func stdNumber(host Host, args []Value) (Value, error) {
	if host.IsNumber(args[0]) {
		return args[0], nil
	}
	text, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("value must be a number or string")
	}
	return parseNumber(text)
}

func stdPanic(host Host, args []Value) (Value, error) {
	return nil, errors.New(host.Stringify(args[0]))
}

func stdPrint(host Host, args []Value) (Value, error) {
	writeValues(host.Stdout(), host, args, false)
	return nil, nil
}

func stdPrintln(host Host, args []Value) (Value, error) {
	writeValues(host.Stdout(), host, args, true)
	return nil, nil
}

func stdRange(host Host, args []Value) (Value, error) {
	if len(args) < 1 || len(args) > 3 {
		return nil, fmt.Errorf("std.range expects 1 to 3 arguments but got %d", len(args))
	}
	var start int64
	end, err := intArg(host, args, 0, "end")
	if err != nil {
		return nil, err
	}
	step := int64(1)
	if len(args) >= 2 {
		start = end
		end, err = intArg(host, args, 1, "end")
		if err != nil {
			return nil, err
		}
	}
	if len(args) == 3 {
		step, err = intArg(host, args, 2, "step")
		if err != nil {
			return nil, err
		}
		if step == 0 {
			return nil, fmt.Errorf("step must not be zero")
		}
	}
	items := make([]Value, 0)
	if step > 0 {
		for i := start; i < end; i += step {
			items = append(items, i)
		}
	} else {
		for i := start; i > end; i += step {
			items = append(items, i)
		}
	}
	return host.NewList(items), nil
}

func stdString(host Host, args []Value) (Value, error) {
	return host.Stringify(args[0]), nil
}

func stdTypeOf(host Host, args []Value) (Value, error) {
	return host.TypeName(args[0]), nil
}

func stdValues(host Host, args []Value) (Value, error) {
	values, ok := keyedValues(host, args[0])
	if !ok {
		return nil, fmt.Errorf("value must be a map or module")
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	items := make([]Value, len(keys))
	for i, key := range keys {
		items[i] = values[key]
	}
	return host.NewList(items), nil
}

func typePredicate(name string) NativeFunc {
	return func(host Host, args []Value) (Value, error) {
		return host.TypeName(args[0]) == name, nil
	}
}

func keyedValues(host Host, value Value) (map[string]Value, bool) {
	if values, ok := host.MapItems(value); ok {
		return values, true
	}
	return host.ModuleExports(value)
}

func writeValues(out interface{ Write([]byte) (int, error) }, host Host, args []Value, newline bool) {
	for i, arg := range args {
		if i > 0 {
			_, _ = fmt.Fprint(out, " ")
		}
		_, _ = fmt.Fprint(out, host.Stringify(arg))
	}
	if newline {
		_, _ = fmt.Fprintln(out)
	}
}
