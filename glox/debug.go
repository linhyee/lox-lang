package glox

import (
	"fmt"
	"io"
)

type OperandKind int

const (
	OperandNone OperandKind = iota
	OperandByte
	OperandConstant
	OperandJump
	OperandLoop
	OperandInvoke
	OperandClosure
)

type InstructionInfo struct {
	Name    string
	Operand OperandKind
}

type Disassembler struct {
	w io.Writer
}

func NewDisassembler(w io.Writer) *Disassembler {
	return &Disassembler{w: w}
}

func (d *Disassembler) Chunk(chunk *Chunk, name string) {
	if d == nil || d.w == nil {
		return
	}
	fmt.Fprintf(d.w, "== %s ==\n", name)
	for offset := 0; offset < len(chunk.Code); {
		offset = d.Instruction(chunk, offset)
	}
}

func (d *Disassembler) Function(function *Function) {
	if function == nil {
		return
	}
	name := function.Name
	if name == "" {
		name = "script"
	}
	d.Chunk(&function.Chunk, name)
}

func (d *Disassembler) Instruction(chunk *Chunk, offset int) int {
	if d == nil || d.w == nil {
		return offset + 1
	}
	if offset < 0 || offset >= len(chunk.Code) {
		fmt.Fprintf(d.w, "%04d <invalid offset>\n", offset)
		return len(chunk.Code)
	}

	d.writePrefix(chunk, offset)
	op := OpCode(chunk.Code[offset])
	info, ok := instructionTable[op]
	if !ok {
		fmt.Fprintf(d.w, "UNKNOWN_%d\n", op)
		return offset + 1
	}

	switch info.Operand {
	case OperandNone:
		return d.simple(info.Name, offset)
	case OperandByte:
		return d.byteOperand(info.Name, chunk, offset)
	case OperandConstant:
		return d.constantOperand(info.Name, chunk, offset)
	case OperandJump:
		return d.jumpOperand(info.Name, 1, chunk, offset)
	case OperandLoop:
		return d.jumpOperand(info.Name, -1, chunk, offset)
	case OperandInvoke:
		return d.invokeOperand(info.Name, chunk, offset)
	case OperandClosure:
		return d.closureOperand(info.Name, chunk, offset)
	default:
		fmt.Fprintf(d.w, "%s <unknown operand kind>\n", info.Name)
		return offset + 1
	}
}

func (d *Disassembler) Stack(values []Value) {
	if d == nil || d.w == nil {
		return
	}
	fmt.Fprint(d.w, "          ")
	for _, value := range values {
		fmt.Fprintf(d.w, "[ %s ]", Stringify(value))
	}
	fmt.Fprintln(d.w)
}

func DisassembleChunk(w io.Writer, chunk *Chunk, name string) {
	NewDisassembler(w).Chunk(chunk, name)
}

func DisassembleInstruction(w io.Writer, chunk *Chunk, offset int) int {
	return NewDisassembler(w).Instruction(chunk, offset)
}

func (d *Disassembler) writePrefix(chunk *Chunk, offset int) {
	fmt.Fprintf(d.w, "%04d ", offset)
	if offset > 0 && chunk.Line(offset) == chunk.Line(offset-1) {
		fmt.Fprint(d.w, "   | ")
		return
	}
	fmt.Fprintf(d.w, "%4d ", chunk.Line(offset))
}

func (d *Disassembler) simple(name string, offset int) int {
	fmt.Fprintln(d.w, name)
	return offset + 1
}

func (d *Disassembler) byteOperand(name string, chunk *Chunk, offset int) int {
	operands, ok := chunk.Operands(offset, 1)
	if !ok {
		fmt.Fprintf(d.w, "%-24s <missing operand>\n", name)
		return len(chunk.Code)
	}
	fmt.Fprintf(d.w, "%-24s %4d\n", name, operands[0])
	return offset + 2
}

func (d *Disassembler) constantOperand(name string, chunk *Chunk, offset int) int {
	operands, ok := chunk.Operands(offset, 1)
	if !ok {
		fmt.Fprintf(d.w, "%-24s <missing constant>\n", name)
		return len(chunk.Code)
	}
	constant := int(operands[0])
	fmt.Fprintf(d.w, "%-24s %4d '%s'\n", name, constant, chunk.ConstantString(constant))
	return offset + 2
}

func (d *Disassembler) jumpOperand(name string, sign int, chunk *Chunk, offset int) int {
	operands, ok := chunk.Operands(offset, 2)
	if !ok {
		fmt.Fprintf(d.w, "%-24s <missing jump>\n", name)
		return len(chunk.Code)
	}
	jump := int(uint16(operands[0])<<8 | uint16(operands[1]))
	fmt.Fprintf(d.w, "%-24s %4d -> %d\n", name, offset, offset+3+sign*jump)
	return offset + 3
}

func (d *Disassembler) invokeOperand(name string, chunk *Chunk, offset int) int {
	operands, ok := chunk.Operands(offset, 2)
	if !ok {
		fmt.Fprintf(d.w, "%-24s <missing operands>\n", name)
		return len(chunk.Code)
	}
	constant := int(operands[0])
	argCount := operands[1]
	fmt.Fprintf(d.w, "%-24s (%d args) %4d '%s'\n", name, argCount, constant, chunk.ConstantString(constant))
	return offset + 3
}

func (d *Disassembler) closureOperand(name string, chunk *Chunk, offset int) int {
	operands, ok := chunk.Operands(offset, 1)
	if !ok {
		fmt.Fprintf(d.w, "%-24s <missing function>\n", name)
		return len(chunk.Code)
	}
	constant := int(operands[0])
	fmt.Fprintf(d.w, "%-24s %4d %s\n", name, constant, chunk.ConstantString(constant))

	function, _ := chunk.Constant(constant).(*Function)
	next := offset + 2
	if function == nil {
		return next
	}
	for i := 0; i < function.UpvalueCount; i++ {
		operands, ok := chunk.Operands(next-1, 3)
		if !ok {
			fmt.Fprintf(d.w, "%04d    |                     <missing upvalue>\n", next)
			return len(chunk.Code)
		}
		kind := "upvalue"
		if operands[0] == 1 {
			kind = "local"
		}
		mutability := "var"
		if operands[2] == 1 {
			mutability = "const"
		}
		fmt.Fprintf(d.w, "%04d    |                     %-7s %4d %s\n", next, kind, operands[1], mutability)
		next += 3
	}
	return next
}

var instructionTable = map[OpCode]InstructionInfo{
	OpConstant:          {"OP_CONSTANT", OperandConstant},
	OpNil:               {"OP_NIL", OperandNone},
	OpTrue:              {"OP_TRUE", OperandNone},
	OpFalse:             {"OP_FALSE", OperandNone},
	OpPop:               {"OP_POP", OperandNone},
	OpDup:               {"OP_DUP", OperandNone},
	OpGetLocal:          {"OP_GET_LOCAL", OperandByte},
	OpSetLocal:          {"OP_SET_LOCAL", OperandByte},
	OpGetGlobal:         {"OP_GET_GLOBAL", OperandConstant},
	OpDefineGlobal:      {"OP_DEFINE_GLOBAL", OperandConstant},
	OpDefineGlobalConst: {"OP_DEFINE_GLOBAL_CONST", OperandConstant},
	OpSetGlobal:         {"OP_SET_GLOBAL", OperandConstant},
	OpGetUpvalue:        {"OP_GET_UPVALUE", OperandByte},
	OpSetUpvalue:        {"OP_SET_UPVALUE", OperandByte},
	OpGetProperty:       {"OP_GET_PROPERTY", OperandConstant},
	OpSetProperty:       {"OP_SET_PROPERTY", OperandConstant},
	OpGetSuper:          {"OP_GET_SUPER", OperandConstant},
	OpGetIndex:          {"OP_GET_INDEX", OperandNone},
	OpSetIndex:          {"OP_SET_INDEX", OperandNone},
	OpList:              {"OP_LIST", OperandByte},
	OpMap:               {"OP_MAP", OperandNone},
	OpMapSet:            {"OP_MAP_SET", OperandNone},
	OpEqual:             {"OP_EQUAL", OperandNone},
	OpGreater:           {"OP_GREATER", OperandNone},
	OpLess:              {"OP_LESS", OperandNone},
	OpAdd:               {"OP_ADD", OperandNone},
	OpSubtract:          {"OP_SUBTRACT", OperandNone},
	OpMultiply:          {"OP_MULTIPLY", OperandNone},
	OpDivide:            {"OP_DIVIDE", OperandNone},
	OpNot:               {"OP_NOT", OperandNone},
	OpNegate:            {"OP_NEGATE", OperandNone},
	OpPrint:             {"OP_PRINT", OperandNone},
	OpJump:              {"OP_JUMP", OperandJump},
	OpJumpIfFalse:       {"OP_JUMP_IF_FALSE", OperandJump},
	OpLoop:              {"OP_LOOP", OperandLoop},
	OpCall:              {"OP_CALL", OperandByte},
	OpInvoke:            {"OP_INVOKE", OperandInvoke},
	OpSuperInvoke:       {"OP_SUPER_INVOKE", OperandInvoke},
	OpClosure:           {"OP_CLOSURE", OperandClosure},
	OpCloseUpvalue:      {"OP_CLOSE_UPVALUE", OperandNone},
	OpReturn:            {"OP_RETURN", OperandNone},
	OpClass:             {"OP_CLASS", OperandConstant},
	OpInherit:           {"OP_INHERIT", OperandNone},
	OpMethod:            {"OP_METHOD", OperandConstant},
	OpPopResult:         {"OP_POP_RESULT", OperandNone},
}
