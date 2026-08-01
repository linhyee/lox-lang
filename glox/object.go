package glox

import (
	"fmt"
	"sort"
)

type FunctionType int

const (
	FunctionScript FunctionType = iota
	FunctionFunction
	FunctionInitializer
	FunctionMethod
	FunctionLambda
)

type Function struct {
	Name         string
	Arity        int
	Chunk        Chunk
	UpvalueCount int
	Type         FunctionType
}

func (f *Function) String() string {
	if f.Name == "" {
		return "<script>"
	}
	return "<fn " + f.Name + ">"
}

type UpvalueSpec struct {
	Index   uint8
	IsLocal bool
	IsConst bool
}

type Upvalue struct {
	location int
	closed   Value
	isClosed bool
	next     *Upvalue
	isConst  bool
}

func (u *Upvalue) Get(vm *VM) Value {
	if u.isClosed {
		return u.closed
	}
	return vm.stack[u.location]
}

func (u *Upvalue) Set(vm *VM, value Value) {
	if u.isClosed {
		u.closed = value
		return
	}
	vm.stack[u.location] = value
}

type Closure struct {
	Function *Function
	Upvalues []*Upvalue
	Module   *Module
}

func NewClosure(function *Function, module *Module) *Closure {
	return &Closure{
		Function: function,
		Upvalues: make([]*Upvalue, function.UpvalueCount),
		Module:   module,
	}
}

func (c *Closure) String() string {
	return c.Function.String()
}

type NativeFunc func(vm *VM, args []Value) (Value, error)

type NativeFunction struct {
	Name  string
	Arity int
	Fn    NativeFunc
}

func (n *NativeFunction) String() string {
	if n.Name == "" {
		return "<native fn>"
	}
	return "<native fn " + n.Name + ">"
}

type BoundNativeFunction struct {
	Receiver Value
	Native   *NativeFunction
}

func (b *BoundNativeFunction) String() string {
	return b.Native.String()
}

type Class struct {
	Name       string
	Superclass *Class
	Methods    map[string]Value
}

func (c *Class) String() string {
	return c.Name
}

func (c *Class) FindMethod(name string) Value {
	if method, ok := c.Methods[name]; ok {
		return method
	}
	if c.Superclass != nil {
		return c.Superclass.FindMethod(name)
	}
	return nil
}

type Instance struct {
	Class  *Class
	Fields map[string]Value
}

func NewInstance(class *Class) *Instance {
	return &Instance{Class: class, Fields: map[string]Value{}}
}

func (i *Instance) String() string {
	return i.Class.Name + " instance"
}

type BoundMethod struct {
	Receiver Value
	Method   *Closure
}

func (b *BoundMethod) String() string {
	return b.Method.String()
}

type List struct {
	Items []Value
}

func (l *List) String() string {
	out := "["
	for i, item := range l.Items {
		if i > 0 {
			out += ", "
		}
		out += Stringify(item)
	}
	return out + "]"
}

type Map struct {
	Items map[string]Value
}

func NewMap() *Map {
	return &Map{Items: map[string]Value{}}
}

func (m *Map) String() string {
	out := "{"
	keys := make([]string, 0, len(m.Items))
	for key := range m.Items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for i, key := range keys {
		if i > 0 {
			out += ", "
		}
		out += key + ": " + Stringify(m.Items[key])
	}
	return out + "}"
}

type Binding struct {
	Value Value
	Const bool
}

type Environment struct {
	values map[string]Binding
}

func NewEnvironment() *Environment {
	return &Environment{values: map[string]Binding{}}
}

func (e *Environment) Define(name string, value Value, isConst bool) {
	e.values[name] = Binding{Value: value, Const: isConst}
}

func (e *Environment) GetOwn(name string) (Value, bool) {
	binding, ok := e.values[name]
	return binding.Value, ok
}

func (e *Environment) Binding(name string) (Binding, bool) {
	binding, ok := e.values[name]
	return binding, ok
}

func (e *Environment) Assign(name string, value Value) error {
	binding, ok := e.values[name]
	if !ok {
		return fmt.Errorf("undefined variable '%s'", name)
	}
	if binding.Const {
		return fmt.Errorf("cannot assign to const '%s'", name)
	}
	binding.Value = value
	e.values[name] = binding
	return nil
}

func (e *Environment) ExportedValues(exports map[string]struct{}) map[string]Value {
	out := map[string]Value{}
	for name := range exports {
		if value, ok := e.GetOwn(name); ok {
			out[name] = value
		}
	}
	return out
}
