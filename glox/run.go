package glox

import "fmt"

func (vm *VM) runUntil(targetFrames int) error {
	for {
		frame := &vm.frames[vm.frameCount-1]
		if vm.debugTraceExecution {
			vm.traceExecution(frame)
		}
		readByte := func() byte {
			b := frame.closure.Function.Chunk.Code[frame.ip]
			frame.ip++
			return b
		}
		readShort := func() uint16 {
			frame.ip += 2
			return uint16(frame.closure.Function.Chunk.Code[frame.ip-2])<<8 |
				uint16(frame.closure.Function.Chunk.Code[frame.ip-1])
		}
		readConstant := func() Value {
			return frame.closure.Function.Chunk.Constants[int(readByte())]
		}
		readString := func() (string, error) {
			value := readConstant()
			name, ok := value.(string)
			if !ok {
				return "", vm.runtimeError("constant is not a string")
			}
			return name, nil
		}

		instruction := OpCode(readByte())
		switch instruction {
		case OpConstant:
			vm.push(readConstant())
		case OpNil:
			vm.push(nil)
		case OpTrue:
			vm.push(true)
		case OpFalse:
			vm.push(false)
		case OpPop:
			_ = vm.pop()
		case OpDup:
			vm.push(vm.peek(0))
		case OpGetLocal:
			slot := int(readByte())
			vm.push(vm.stack[frame.slots+slot])
		case OpSetLocal:
			slot := int(readByte())
			vm.stack[frame.slots+slot] = vm.peek(0)
		case OpGetGlobal:
			name, err := readString()
			if err != nil {
				return err
			}
			value, ok := vm.readGlobal(name, frame.closure.Module)
			if !ok {
				return vm.runtimeError("undefined variable '%s'", name)
			}
			vm.push(value)
		case OpDefineGlobal, OpDefineGlobalConst:
			name, err := readString()
			if err != nil {
				return err
			}
			env := vm.globals
			if frame.closure.Module != nil {
				env = frame.closure.Module.Env
			}
			env.Define(name, vm.peek(0), instruction == OpDefineGlobalConst)
			_ = vm.pop()
		case OpSetGlobal:
			name, err := readString()
			if err != nil {
				return err
			}
			if err := vm.assignGlobal(name, vm.peek(0), frame.closure.Module); err != nil {
				return vm.runtimeError("%s", err.Error())
			}
		case OpGetUpvalue:
			slot := int(readByte())
			vm.push(frame.closure.Upvalues[slot].Get(vm))
		case OpSetUpvalue:
			slot := int(readByte())
			upvalue := frame.closure.Upvalues[slot]
			if upvalue.isConst {
				return vm.runtimeError("cannot assign to const captured variable")
			}
			upvalue.Set(vm, vm.peek(0))
		case OpGetProperty:
			name, err := readString()
			if err != nil {
				return err
			}
			if err := vm.getProperty(name); err != nil {
				return err
			}
		case OpSetProperty:
			name, err := readString()
			if err != nil {
				return err
			}
			if err := vm.setProperty(name); err != nil {
				return err
			}
		case OpGetSuper:
			name, err := readString()
			if err != nil {
				return err
			}
			superclass, ok := vm.pop().(*Class)
			if !ok {
				return vm.runtimeError("super must be a class")
			}
			receiver := vm.peek(0)
			if err := vm.bindMethod(receiver, superclass, name); err != nil {
				return err
			}
		case OpGetIndex:
			if err := vm.getIndex(); err != nil {
				return err
			}
		case OpSetIndex:
			if err := vm.setIndex(); err != nil {
				return err
			}
		case OpList:
			length := int(readByte())
			items := append([]Value(nil), vm.stack[vm.stackTop-length:vm.stackTop]...)
			vm.stackTop -= length
			vm.push(&List{Items: items})
		case OpMap:
			vm.push(NewMap())
		case OpMapSet:
			value := vm.pop()
			key, ok := vm.pop().(string)
			if !ok {
				return vm.runtimeError("map key must be a string")
			}
			object, ok := vm.peek(0).(*Map)
			if !ok {
				return vm.runtimeError("map data can only be added to a map")
			}
			object.Items[key] = value
		case OpEqual:
			b := vm.pop()
			a := vm.pop()
			vm.push(ValuesEqual(a, b))
		case OpGreater:
			if err := vm.binaryNumber(func(a, b float64) Value { return a > b }); err != nil {
				return err
			}
		case OpLess:
			if err := vm.binaryNumber(func(a, b float64) Value { return a < b }); err != nil {
				return err
			}
		case OpAdd:
			if err := vm.addValues(); err != nil {
				return err
			}
		case OpSubtract:
			if err := vm.binaryNumber(func(a, b float64) Value { return a - b }); err != nil {
				return err
			}
		case OpMultiply:
			if err := vm.binaryNumber(func(a, b float64) Value { return a * b }); err != nil {
				return err
			}
		case OpDivide:
			if err := vm.binaryNumber(func(a, b float64) Value { return a / b }); err != nil {
				return err
			}
		case OpNot:
			vm.push(IsFalsey(vm.pop()))
		case OpNegate:
			value, ok := vm.pop().(float64)
			if !ok {
				return vm.runtimeError("operand must be a number")
			}
			vm.push(-value)
		case OpPrint:
			fmt.Fprintln(vm.Stdout, Stringify(vm.pop()))
		case OpJump:
			frame.ip += int(readShort())
		case OpJumpIfFalse:
			offset := int(readShort())
			if IsFalsey(vm.peek(0)) {
				frame.ip += offset
			}
		case OpLoop:
			frame.ip -= int(readShort())
		case OpCall:
			argCount := int(readByte())
			if err := vm.callValue(vm.peek(argCount), argCount); err != nil {
				return err
			}
		case OpInvoke:
			name, err := readString()
			if err != nil {
				return err
			}
			argCount := int(readByte())
			if err := vm.invoke(name, argCount); err != nil {
				return err
			}
		case OpSuperInvoke:
			name, err := readString()
			if err != nil {
				return err
			}
			argCount := int(readByte())
			superclass, ok := vm.pop().(*Class)
			if !ok {
				return vm.runtimeError("super must be a class")
			}
			receiver := vm.peek(argCount)
			if err := vm.invokeFromClass(receiver, superclass, name, argCount); err != nil {
				return err
			}
		case OpClosure:
			function, ok := readConstant().(*Function)
			if !ok {
				return vm.runtimeError("closure constant is not a function")
			}
			closure := NewClosure(function, frame.closure.Module)
			vm.push(closure)
			for i := 0; i < function.UpvalueCount; i++ {
				isLocal := readByte() == 1
				index := int(readByte())
				isConst := readByte() == 1
				if isLocal {
					closure.Upvalues[i] = vm.captureUpvalue(frame.slots+index, isConst)
				} else {
					closure.Upvalues[i] = frame.closure.Upvalues[index]
				}
			}
		case OpCloseUpvalue:
			vm.closeUpvalues(vm.stackTop - 1)
			_ = vm.pop()
		case OpReturn:
			result := vm.pop()
			if frame.closure.Function.Type == FunctionInitializer {
				result = vm.stack[frame.slots]
			}
			vm.closeUpvalues(frame.slots)
			vm.frameCount--
			vm.stackTop = frame.slots
			vm.push(result)
			if vm.frameCount == targetFrames {
				return nil
			}
		case OpClass:
			name, err := readString()
			if err != nil {
				return err
			}
			vm.push(&Class{Name: name, Methods: map[string]Value{}})
		case OpInherit:
			superclass, ok := vm.peek(1).(*Class)
			if !ok {
				return vm.runtimeError("superclass must be a class")
			}
			subclass, ok := vm.peek(0).(*Class)
			if !ok {
				return vm.runtimeError("subclass must be a class")
			}
			subclass.Superclass = superclass
			for name, method := range superclass.Methods {
				subclass.Methods[name] = method
			}
			_ = vm.pop()
		case OpMethod:
			name, err := readString()
			if err != nil {
				return err
			}
			if err := vm.defineMethod(name); err != nil {
				return err
			}
		default:
			return vm.runtimeError("unknown opcode %d", instruction)
		}
	}
}

func (vm *VM) traceExecution(frame *CallFrame) {
	if vm.debug == nil || frame == nil || frame.closure == nil {
		return
	}
	vm.debug.Stack(vm.stack[:vm.stackTop])
	vm.debug.Instruction(&frame.closure.Function.Chunk, frame.ip)
}

func (vm *VM) getProperty(name string) error {
	receiver := vm.peek(0)
	switch object := receiver.(type) {
	case *Instance:
		if value, ok := object.Fields[name]; ok {
			_ = vm.pop()
			vm.push(value)
			return nil
		}
		return vm.bindMethod(object, object.Class, name)
	case *List:
		method := vm.listMethod(object, name)
		if method == nil {
			return vm.runtimeError("undefined property '%s'", name)
		}
		_ = vm.pop()
		vm.push(method)
		return nil
	case *Map:
		if value, ok := object.Items[name]; ok {
			_ = vm.pop()
			vm.push(value)
			return nil
		}
		method := vm.mapMethod(object, name)
		if method == nil {
			return vm.runtimeError("undefined property '%s'", name)
		}
		_ = vm.pop()
		vm.push(method)
		return nil
	case *Module:
		value, err := vm.getModuleProperty(object, name)
		if err != nil {
			return vm.runtimeError("%s", err.Error())
		}
		_ = vm.pop()
		vm.push(value)
		return nil
	default:
		return vm.runtimeError("only instances, lists, maps and modules have properties")
	}
}

func (vm *VM) setProperty(name string) error {
	value := vm.peek(0)
	receiver := vm.peek(1)
	switch object := receiver.(type) {
	case *Instance:
		object.Fields[name] = value
	case *Module:
		if err := vm.setModuleProperty(object, name, value); err != nil {
			return vm.runtimeError("%s", err.Error())
		}
	case *Map:
		object.Items[name] = value
	default:
		return vm.runtimeError("only instances, maps and modules have fields")
	}
	assigned := vm.pop()
	_ = vm.pop()
	vm.push(assigned)
	return nil
}

func (vm *VM) getIndex() error {
	index := vm.pop()
	collection := vm.pop()
	switch object := collection.(type) {
	case *List:
		i, ok := numberToIndex(index, len(object.Items))
		if !ok {
			return vm.runtimeError("list index out of range")
		}
		vm.push(object.Items[i])
	case string:
		i, ok := numberToIndex(index, len([]rune(object)))
		if !ok {
			return vm.runtimeError("string index out of range")
		}
		vm.push(string([]rune(object)[i]))
	case *Map:
		key, ok := index.(string)
		if !ok {
			return vm.runtimeError("map key must be a string")
		}
		vm.push(object.Items[key])
	default:
		return vm.runtimeError("can only index lists, maps and strings")
	}
	return nil
}

func (vm *VM) setIndex() error {
	value := vm.pop()
	index := vm.pop()
	collection := vm.pop()
	switch object := collection.(type) {
	case *List:
		i, ok := numberToIndex(index, len(object.Items))
		if !ok {
			return vm.runtimeError("list index out of range")
		}
		object.Items[i] = value
	case *Map:
		key, ok := index.(string)
		if !ok {
			return vm.runtimeError("map key must be a string")
		}
		object.Items[key] = value
	default:
		return vm.runtimeError("can only assign list or map elements")
	}
	vm.push(value)
	return nil
}

func (vm *VM) binaryNumber(op func(float64, float64) Value) error {
	right, okRight := vm.pop().(float64)
	left, okLeft := vm.pop().(float64)
	if !okLeft || !okRight {
		return vm.runtimeError("operands must be numbers")
	}
	vm.push(op(left, right))
	return nil
}

func (vm *VM) addValues() error {
	right := vm.pop()
	left := vm.pop()
	if a, ok := left.(float64); ok {
		if b, ok := right.(float64); ok {
			vm.push(a + b)
			return nil
		}
	}
	if a, ok := left.(string); ok {
		if b, ok := right.(string); ok {
			vm.push(a + b)
			return nil
		}
	}
	return vm.runtimeError("operands must be two numbers or two strings")
}
