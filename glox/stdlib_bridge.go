package glox

import (
	"io"

	"glox/stdlib"
)

func (vm *VM) defineStdlibModules() {
	for name, module := range stdlib.Modules() {
		registered := vm.RegisterNativeModule(name, vm.convertStdlibModule(module))
		vm.Loader.RegisterNativeModuleAlias(name+".lox", registered)
		vm.Loader.RegisterNativeModuleAlias("stdlib/"+name, registered)
		vm.Loader.RegisterNativeModuleAlias("stdlib/"+name+".lox", registered)
	}
}

func (vm *VM) convertStdlibModule(module stdlib.Module) map[string]Value {
	host := stdlibHost{vm: vm}
	exports := make(map[string]Value, len(module))
	for name, value := range module {
		switch native := value.(type) {
		case *stdlib.Native:
			nativeValue := native
			exports[name] = &NativeFunction{Name: nativeValue.Name, Arity: nativeValue.Arity, Fn: func(vm *VM, args []Value) (Value, error) {
				stdArgs := make([]stdlib.Value, len(args))
				for i, arg := range args {
					stdArgs[i] = arg
				}
				return nativeValue.Fn(host.withVM(vm), stdArgs)
			}}
		default:
			exports[name] = value
		}
	}
	return exports
}

type stdlibHost struct {
	vm *VM
}

func (h stdlibHost) withVM(vm *VM) stdlibHost {
	h.vm = vm
	return h
}

func (h stdlibHost) Args() []string {
	return append([]string(nil), h.vm.Args...)
}

func (h stdlibHost) Stdout() io.Writer {
	return h.vm.Stdout
}

func (h stdlibHost) Stderr() io.Writer {
	return h.vm.Stderr
}

func (h stdlibHost) Stringify(value stdlib.Value) string {
	return Stringify(value)
}

func (h stdlibHost) TypeName(value stdlib.Value) string {
	return typeName(value)
}

func (h stdlibHost) IsNumber(value stdlib.Value) bool {
	return IsNumber(value)
}

func (h stdlibHost) AsInt64(value stdlib.Value) (int64, bool) {
	return AsInt64(value)
}

func (h stdlibHost) AsFloat64(value stdlib.Value) (float64, bool) {
	return AsFloat64(value)
}

func (h stdlibHost) NewList(items []stdlib.Value) stdlib.Value {
	values := make([]Value, len(items))
	for i, item := range items {
		values[i] = item
	}
	return &List{Items: values}
}

func (h stdlibHost) ListItems(value stdlib.Value) ([]stdlib.Value, bool) {
	list, ok := value.(*List)
	if !ok {
		return nil, false
	}
	items := make([]stdlib.Value, len(list.Items))
	for i, item := range list.Items {
		items[i] = item
	}
	return items, true
}

func (h stdlibHost) NewMap(items map[string]stdlib.Value) stdlib.Value {
	values := make(map[string]Value, len(items))
	for key, value := range items {
		values[key] = value
	}
	return &Map{Items: values}
}

func (h stdlibHost) MapItems(value stdlib.Value) (map[string]stdlib.Value, bool) {
	object, ok := value.(*Map)
	if !ok {
		return nil, false
	}
	items := make(map[string]stdlib.Value, len(object.Items))
	for key, item := range object.Items {
		items[key] = item
	}
	return items, true
}

func (h stdlibHost) ModuleExports(value stdlib.Value) (map[string]stdlib.Value, bool) {
	module, ok := value.(*Module)
	if !ok {
		return nil, false
	}
	values := module.Env.ExportedValues(module.Exports)
	items := make(map[string]stdlib.Value, len(values))
	for key, item := range values {
		items[key] = item
	}
	return items, true
}
