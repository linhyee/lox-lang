package glox

import (
	"fmt"
	"time"
	"unicode/utf8"
)

func (vm *VM) DefineNative(name string, arity int, fn NativeFunc) {
	vm.globals.Define(name, &NativeFunction{Name: name, Arity: arity, Fn: fn}, true)
}

func (vm *VM) DefineValue(name string, value Value, isConst bool) {
	vm.globals.Define(name, value, isConst)
}

func (vm *VM) RegisterNativeModule(name string, exports map[string]Value) *Module {
	return vm.Loader.RegisterNativeModule(name, exports)
}

func (vm *VM) defineBuiltins() {
	vm.DefineNative("clock", 0, func(vm *VM, args []Value) (Value, error) {
		return float64(time.Now().UnixNano()) / 1e9, nil
	})
	vm.DefineNative("len", 1, func(vm *VM, args []Value) (Value, error) {
		switch v := args[0].(type) {
		case string:
			return int64(utf8.RuneCountInString(v)), nil
		case *List:
			return int64(len(v.Items)), nil
		case *Map:
			return int64(len(v.Items)), nil
		case *Module:
			return int64(len(v.Exports)), nil
		default:
			return int64(0), nil
		}
	})
	vm.DefineNative("type", 1, func(vm *VM, args []Value) (Value, error) {
		return typeName(args[0]), nil
	})
	vm.DefineNative("string", 1, func(vm *VM, args []Value) (Value, error) {
		return Stringify(args[0]), nil
	})
	vm.DefineNative("import", 1, func(vm *VM, args []Value) (Value, error) {
		path, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("import path must be a string")
		}
		caller := vm.currentModule()
		line := 0
		if vm.frameCount > 0 {
			frame := &vm.frames[vm.frameCount-1]
			if frame.ip > 0 && frame.ip-1 < len(frame.closure.Function.Chunk.Lines) {
				line = frame.closure.Function.Chunk.Lines[frame.ip-1]
			}
		}
		return vm.Loader.Import(path, caller, line)
	})
	vm.defineStdlibModules()
}

func typeName(value Value) string {
	switch value.(type) {
	case nil:
		return "nil"
	case bool:
		return "boolean"
	case float64:
		return "number"
	case int64:
		return "number"
	case string:
		return "string"
	case *Function, *Closure, *BoundMethod:
		return "function"
	case *NativeFunction, *BoundNativeFunction:
		return "native-function"
	case *Class:
		return "class"
	case *Instance:
		return "object"
	case *List:
		return "list"
	case *Map:
		return "map"
	case *Module:
		return "module"
	default:
		return "unknown"
	}
}

func (vm *VM) mapMethod(receiver *Map, name string) Value {
	switch name {
	case "size":
		return &BoundNativeFunction{Receiver: receiver, Native: &NativeFunction{Name: "Map.size", Arity: 0, Fn: func(vm *VM, args []Value) (Value, error) {
			return int64(len(args[0].(*Map).Items)), nil
		}}}
	case "keys":
		return &BoundNativeFunction{Receiver: receiver, Native: &NativeFunction{Name: "Map.keys", Arity: 0, Fn: func(vm *VM, args []Value) (Value, error) {
			items := make([]Value, 0, len(args[0].(*Map).Items))
			for key := range args[0].(*Map).Items {
				items = append(items, key)
			}
			return &List{Items: items}, nil
		}}}
	case "has":
		return &BoundNativeFunction{Receiver: receiver, Native: &NativeFunction{Name: "Map.has", Arity: 1, Fn: func(vm *VM, args []Value) (Value, error) {
			key, ok := args[1].(string)
			if !ok {
				return nil, fmt.Errorf("map key must be a string")
			}
			_, exists := args[0].(*Map).Items[key]
			return exists, nil
		}}}
	case "remove":
		return &BoundNativeFunction{Receiver: receiver, Native: &NativeFunction{Name: "Map.remove", Arity: 1, Fn: func(vm *VM, args []Value) (Value, error) {
			key, ok := args[1].(string)
			if !ok {
				return nil, fmt.Errorf("map key must be a string")
			}
			value := args[0].(*Map).Items[key]
			delete(args[0].(*Map).Items, key)
			return value, nil
		}}}
	default:
		return nil
	}
}

func (vm *VM) listMethod(receiver *List, name string) Value {
	switch name {
	case "push":
		return &BoundNativeFunction{Receiver: receiver, Native: &NativeFunction{Name: "List.push", Arity: 1, Fn: func(vm *VM, args []Value) (Value, error) {
			list := args[0].(*List)
			list.Items = append(list.Items, args[1])
			return true, nil
		}}}
	case "pop":
		return &BoundNativeFunction{Receiver: receiver, Native: &NativeFunction{Name: "List.pop", Arity: 0, Fn: func(vm *VM, args []Value) (Value, error) {
			list := args[0].(*List)
			if len(list.Items) == 0 {
				return nil, nil
			}
			value := list.Items[len(list.Items)-1]
			list.Items = list.Items[:len(list.Items)-1]
			return value, nil
		}}}
	case "size":
		return &BoundNativeFunction{Receiver: receiver, Native: &NativeFunction{Name: "List.size", Arity: 0, Fn: func(vm *VM, args []Value) (Value, error) {
			return int64(len(args[0].(*List).Items)), nil
		}}}
	case "insertAt":
		return &BoundNativeFunction{Receiver: receiver, Native: &NativeFunction{Name: "List.insertAt", Arity: 2, Fn: func(vm *VM, args []Value) (Value, error) {
			list := args[0].(*List)
			index, ok := numberToIndex(args[1], len(list.Items)+1)
			if !ok {
				return nil, fmt.Errorf("list index must be a valid integer")
			}
			value := args[2]
			list.Items = append(list.Items, nil)
			copy(list.Items[index+1:], list.Items[index:])
			list.Items[index] = value
			return true, nil
		}}}
	case "remove":
		return &BoundNativeFunction{Receiver: receiver, Native: &NativeFunction{Name: "List.remove", Arity: 1, Fn: func(vm *VM, args []Value) (Value, error) {
			list := args[0].(*List)
			index, ok := numberToIndex(args[1], len(list.Items))
			if !ok {
				return nil, fmt.Errorf("list index out of range")
			}
			value := list.Items[index]
			list.Items = append(list.Items[:index], list.Items[index+1:]...)
			return value, nil
		}}}
	default:
		return nil
	}
}
