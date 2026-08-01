package glox

import "fmt"

type OpCode byte

const (
	OpConstant OpCode = iota
	OpNil
	OpTrue
	OpFalse
	OpPop
	OpDup
	OpGetLocal
	OpSetLocal
	OpGetGlobal
	OpDefineGlobal
	OpDefineGlobalConst
	OpSetGlobal
	OpGetUpvalue
	OpSetUpvalue
	OpGetProperty
	OpSetProperty
	OpGetSuper
	OpGetIndex
	OpSetIndex
	OpList
	OpMap
	OpMapSet
	OpEqual
	OpGreater
	OpLess
	OpAdd
	OpSubtract
	OpMultiply
	OpDivide
	OpNot
	OpNegate
	OpPrint
	OpJump
	OpJumpIfFalse
	OpLoop
	OpCall
	OpInvoke
	OpSuperInvoke
	OpClosure
	OpCloseUpvalue
	OpReturn
	OpClass
	OpInherit
	OpMethod
)

type Chunk struct {
	Code      []byte
	Lines     []int
	Constants []Value
}

func (c *Chunk) Write(b byte, line int) {
	c.Code = append(c.Code, b)
	c.Lines = append(c.Lines, line)
}

func (c *Chunk) AddConstant(value Value) int {
	c.Constants = append(c.Constants, value)
	return len(c.Constants) - 1
}

func (c *Chunk) Constant(index int) Value {
	if index < 0 || index >= len(c.Constants) {
		return nil
	}
	return c.Constants[index]
}

func (c *Chunk) ConstantString(index int) string {
	if index < 0 || index >= len(c.Constants) {
		return "<invalid constant>"
	}
	return Stringify(c.Constants[index])
}

func (c *Chunk) Line(offset int) int {
	if offset < 0 || offset >= len(c.Lines) {
		return 0
	}
	return c.Lines[offset]
}

func (c *Chunk) Operands(offset int, count int) ([]byte, bool) {
	start := offset + 1
	end := start + count
	if offset < 0 || count < 0 || end > len(c.Code) {
		return nil, false
	}
	return c.Code[start:end], true
}

func (op OpCode) String() string {
	if info, ok := instructionTable[op]; ok {
		return info.Name
	}
	return fmt.Sprintf("OpCode(%d)", op)
}
