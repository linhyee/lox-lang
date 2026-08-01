package glox

import (
	"fmt"
	"io"
	"os"
)

const (
	framesMax = 64
	stackMax  = framesMax * 256
)

type Options struct {
	Stdout              io.Writer
	Stderr              io.Writer
	RootDir             string
	ModulePaths         []string
	DebugWriter         io.Writer
	DebugDisassemble    bool
	DebugTraceExecution bool
}

type CallFrame struct {
	closure *Closure
	ip      int
	slots   int
}

type VM struct {
	stack       []Value
	stackTop    int
	frames      []CallFrame
	frameCount  int
	openUpvalue *Upvalue

	globals     *Environment
	replModule  *Module
	Loader      *ModuleLoader
	Diagnostics *Diagnostics
	Stdout      io.Writer
	Stderr      io.Writer
	debug       *Disassembler

	debugDisassemble    bool
	debugTraceExecution bool
}

func NewVM(options Options) *VM {
	stdout := options.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := options.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	debugWriter := options.DebugWriter
	if debugWriter == nil {
		debugWriter = stderr
	}
	vm := &VM{
		stack:       make([]Value, stackMax),
		frames:      make([]CallFrame, framesMax),
		globals:     NewEnvironment(),
		Diagnostics: NewDiagnostics(stderr),
		Stdout:      stdout,
		Stderr:      stderr,
		debug:       NewDisassembler(debugWriter),

		debugDisassemble:    options.DebugDisassemble,
		debugTraceExecution: options.DebugTraceExecution,
	}
	vm.Loader = NewModuleLoader(vm, options.RootDir, options.ModulePaths)
	vm.defineBuiltins()
	return vm
}

func (vm *VM) RunString(source string) error {
	return vm.Interpret(source, "repl")
}

func (vm *VM) RunFile(path string) error {
	source, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return vm.Interpret(string(source), path)
}

func (vm *VM) Interpret(source, path string) error {
	vm.Diagnostics.Reset()
	vm.resetStack()
	resolved := path
	var module *Module
	if path == "repl" {
		if vm.replModule == nil {
			vm.replModule = NewModule("repl")
			vm.replModule.State = ModuleInitialized
			vm.Loader.loaded["repl"] = vm.replModule
		}
		module = vm.replModule
	} else {
		module = NewModule(path)
		if abs, err := vm.Loader.Resolve(path, nil); err == nil {
			resolved = abs
			module.Path = abs
			vm.Loader.loaded[abs] = module
			vm.Loader.loadingStack = append(vm.Loader.loadingStack, abs)
			defer func() {
				if len(vm.Loader.loadingStack) > 0 && vm.Loader.loadingStack[len(vm.Loader.loadingStack)-1] == abs {
					vm.Loader.loadingStack = vm.Loader.loadingStack[:len(vm.Loader.loadingStack)-1]
				}
			}()
		}
	}
	function, err := Compile(source, resolved, module, vm.Diagnostics)
	if err != nil {
		module.State = ModuleFailed
		return err
	}
	vm.disassembleCompiledFunction(function)
	module.Closure = NewClosure(function, module)
	module.State = ModuleReady
	if err := vm.executeModule(module); err != nil {
		module.State = ModuleFailed
		return err
	}
	return nil
}

func (vm *VM) disassembleCompiledFunction(function *Function) {
	if !vm.debugDisassemble || vm.debug == nil {
		return
	}
	seen := map[*Function]bool{}
	var walk func(*Function)
	walk = func(fn *Function) {
		if fn == nil || seen[fn] {
			return
		}
		seen[fn] = true
		vm.debug.Function(fn)
		for _, constant := range fn.Chunk.Constants {
			if child, ok := constant.(*Function); ok {
				walk(child)
			}
		}
	}
	walk(function)
}

func (vm *VM) executeModule(module *Module) error {
	if module.State == ModuleInitialized {
		vm.push(module)
		return nil
	}
	if module.State == ModuleExecuting {
		return fmt.Errorf("circular import detected while executing module '%s'", module.Path)
	}
	module.State = ModuleExecuting
	baseFrames := vm.frameCount
	vm.push(module.Closure)
	if err := vm.call(module.Closure, 0); err != nil {
		return err
	}
	if err := vm.runUntil(baseFrames); err != nil {
		return err
	}
	if vm.stackTop > 0 {
		_ = vm.pop()
	}
	module.State = ModuleInitialized
	vm.push(module)
	return nil
}

func (vm *VM) resetStack() {
	vm.stackTop = 0
	vm.frameCount = 0
	vm.openUpvalue = nil
}

func (vm *VM) push(value Value) {
	vm.stack[vm.stackTop] = value
	vm.stackTop++
}

func (vm *VM) pop() Value {
	vm.stackTop--
	value := vm.stack[vm.stackTop]
	vm.stack[vm.stackTop] = nil
	return value
}

func (vm *VM) peek(distance int) Value {
	return vm.stack[vm.stackTop-1-distance]
}

func (vm *VM) currentModule() *Module {
	if vm.frameCount == 0 {
		return nil
	}
	return vm.frames[vm.frameCount-1].closure.Module
}

func (vm *VM) runtimeError(format string, args ...any) error {
	message := fmt.Sprintf(format, args...)
	for i := vm.frameCount - 1; i >= 0; i-- {
		frame := &vm.frames[i]
		function := frame.closure.Function
		instruction := frame.ip - 1
		line := 0
		if instruction >= 0 && instruction < len(function.Chunk.Lines) {
			line = function.Chunk.Lines[instruction]
		}
		name := function.Name
		if name == "" {
			name = "script"
		} else {
			name += "()"
		}
		message += fmt.Sprintf("\n[line %d] in %s", line, name)
	}
	vm.Diagnostics.Runtime(message)
	vm.resetStack()
	return &RuntimeError{Message: message}
}

func (vm *VM) call(closure *Closure, argCount int) error {
	if argCount != closure.Function.Arity {
		return vm.runtimeError("expected %d arguments but got %d", closure.Function.Arity, argCount)
	}
	if vm.frameCount == framesMax {
		return vm.runtimeError("stack overflow")
	}
	frame := &vm.frames[vm.frameCount]
	vm.frameCount++
	frame.closure = closure
	frame.ip = 0
	frame.slots = vm.stackTop - argCount - 1
	return nil
}

func (vm *VM) callValue(callee Value, argCount int) error {
	calleeIndex := vm.stackTop - argCount - 1
	switch fn := callee.(type) {
	case *Closure:
		return vm.call(fn, argCount)
	case *Class:
		instance := NewInstance(fn)
		vm.stack[calleeIndex] = instance
		initializer := fn.FindMethod("init")
		if initializer == nil {
			if argCount != 0 {
				return vm.runtimeError("expected 0 arguments but got %d", argCount)
			}
			return nil
		}
		return vm.call(initializer.(*Closure), argCount)
	case *BoundMethod:
		vm.stack[calleeIndex] = fn.Receiver
		return vm.call(fn.Method, argCount)
	case *NativeFunction:
		if err := vm.checkNativeArity(fn, argCount); err != nil {
			return err
		}
		args := append([]Value(nil), vm.stack[calleeIndex+1:vm.stackTop]...)
		result, err := fn.Fn(vm, args)
		if err != nil {
			return vm.runtimeError("%s", err.Error())
		}
		vm.stackTop = calleeIndex
		vm.push(result)
		return nil
	case *BoundNativeFunction:
		if err := vm.checkNativeArity(fn.Native, argCount); err != nil {
			return err
		}
		args := append([]Value{fn.Receiver}, vm.stack[calleeIndex+1:vm.stackTop]...)
		result, err := fn.Native.Fn(vm, args)
		if err != nil {
			return vm.runtimeError("%s", err.Error())
		}
		vm.stackTop = calleeIndex
		vm.push(result)
		return nil
	default:
		return vm.runtimeError("can only call functions and classes")
	}
}

func (vm *VM) checkNativeArity(fn *NativeFunction, argCount int) error {
	if fn.Arity >= 0 && argCount != fn.Arity {
		return vm.runtimeError("expected %d arguments but got %d", fn.Arity, argCount)
	}
	return nil
}

func (vm *VM) captureUpvalue(location int, isConst bool) *Upvalue {
	var prev *Upvalue
	current := vm.openUpvalue
	for current != nil && current.location > location {
		prev = current
		current = current.next
	}
	if current != nil && current.location == location {
		return current
	}
	created := &Upvalue{location: location, next: current, isConst: isConst}
	if prev == nil {
		vm.openUpvalue = created
	} else {
		prev.next = created
	}
	return created
}

func (vm *VM) closeUpvalues(last int) {
	for vm.openUpvalue != nil && vm.openUpvalue.location >= last {
		upvalue := vm.openUpvalue
		upvalue.closed = vm.stack[upvalue.location]
		upvalue.isClosed = true
		vm.openUpvalue = upvalue.next
	}
}

func (vm *VM) defineMethod(name string) error {
	method := vm.peek(0)
	class, ok := vm.peek(1).(*Class)
	if !ok {
		return vm.runtimeError("method target must be a class")
	}
	class.Methods[name] = method
	_ = vm.pop()
	return nil
}

func (vm *VM) bindMethod(receiver Value, class *Class, name string) error {
	method := class.FindMethod(name)
	if method == nil {
		return vm.runtimeError("undefined property '%s'", name)
	}
	closure, ok := method.(*Closure)
	if !ok {
		return vm.runtimeError("method '%s' is not callable", name)
	}
	_ = vm.pop()
	vm.push(&BoundMethod{Receiver: receiver, Method: closure})
	return nil
}

func (vm *VM) invoke(name string, argCount int) error {
	receiver := vm.peek(argCount)
	switch r := receiver.(type) {
	case *Instance:
		if value, ok := r.Fields[name]; ok {
			vm.stack[vm.stackTop-argCount-1] = value
			return vm.callValue(value, argCount)
		}
		return vm.invokeFromClass(r, r.Class, name, argCount)
	case *List:
		method := vm.listMethod(r, name)
		if method == nil {
			return vm.runtimeError("undefined property '%s'", name)
		}
		vm.stack[vm.stackTop-argCount-1] = method
		return vm.callValue(method, argCount)
	case *Map:
		method := vm.mapMethod(r, name)
		if method == nil {
			return vm.runtimeError("undefined property '%s'", name)
		}
		vm.stack[vm.stackTop-argCount-1] = method
		return vm.callValue(method, argCount)
	case *Module:
		value, err := vm.getModuleProperty(r, name)
		if err != nil {
			return vm.runtimeError("%s", err.Error())
		}
		vm.stack[vm.stackTop-argCount-1] = value
		return vm.callValue(value, argCount)
	default:
		return vm.runtimeError("only instances, lists, maps and modules have methods")
	}
}

func (vm *VM) invokeFromClass(receiver Value, class *Class, name string, argCount int) error {
	method := class.FindMethod(name)
	if method == nil {
		return vm.runtimeError("undefined property '%s'", name)
	}
	closure, ok := method.(*Closure)
	if !ok {
		return vm.runtimeError("method '%s' is not callable", name)
	}
	vm.stack[vm.stackTop-argCount-1] = receiver
	return vm.call(closure, argCount)
}

func (vm *VM) getModuleProperty(module *Module, name string) (Value, error) {
	if module.State != ModuleInitialized {
		return nil, fmt.Errorf("module '%s' is not initialized", module.Path)
	}
	if !module.IsExported(name) {
		return nil, fmt.Errorf("undefined export '%s' in module '%s'", name, module.Path)
	}
	value, ok := module.Env.GetOwn(name)
	if !ok {
		return nil, fmt.Errorf("export '%s' is not defined in module '%s'", name, module.Path)
	}
	return value, nil
}

func (vm *VM) setModuleProperty(module *Module, name string, value Value) error {
	if module.State != ModuleInitialized {
		return fmt.Errorf("module '%s' is not initialized", module.Path)
	}
	if !module.IsExported(name) {
		return fmt.Errorf("cannot set non-exported property '%s' in module '%s'", name, module.Path)
	}
	return module.Env.Assign(name, value)
}

func (vm *VM) readGlobal(name string, module *Module) (Value, bool) {
	if module != nil {
		if value, ok := module.Env.GetOwn(name); ok {
			return value, true
		}
	}
	return vm.globals.GetOwn(name)
}

func (vm *VM) assignGlobal(name string, value Value, module *Module) error {
	if module != nil {
		if _, ok := module.Env.Binding(name); ok {
			return module.Env.Assign(name, value)
		}
	}
	return vm.globals.Assign(name, value)
}

func numberToIndex(value Value, limit int) (int, bool) {
	number, ok := value.(float64)
	if !ok {
		return 0, false
	}
	index := int(number)
	if float64(index) != number || index < 0 || index >= limit {
		return 0, false
	}
	return index, true
}
