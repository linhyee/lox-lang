package glox

import "testing"

func TestChunkWriteConstantsAndHelpers(t *testing.T) {
	var chunk Chunk
	hello := chunk.AddConstant("hello")
	number := chunk.AddConstant(float64(3.5))
	chunk.Write(byte(OpConstant), 7)
	chunk.Write(byte(hello), 7)
	chunk.Write(byte(OpReturn), 8)

	if hello != 0 || number != 1 {
		t.Fatalf("unexpected constant indexes: %d %d", hello, number)
	}
	if got := chunk.Constant(hello); got != "hello" {
		t.Fatalf("constant mismatch: %#v", got)
	}
	if got := chunk.ConstantString(number); got != "3.5" {
		t.Fatalf("constant string mismatch: %q", got)
	}
	if got := chunk.Constant(-1); got != nil {
		t.Fatalf("invalid constant should be nil: %#v", got)
	}
	if got := chunk.ConstantString(9); got != "<invalid constant>" {
		t.Fatalf("invalid constant string mismatch: %q", got)
	}
	if got := chunk.Line(0); got != 7 {
		t.Fatalf("line mismatch: %d", got)
	}
	if got := chunk.Line(99); got != 0 {
		t.Fatalf("invalid line mismatch: %d", got)
	}
	if operands, ok := chunk.Operands(0, 1); !ok || len(operands) != 1 || operands[0] != byte(hello) {
		t.Fatalf("operands mismatch: %#v ok=%v", operands, ok)
	}
	if _, ok := chunk.Operands(1, 2); ok {
		t.Fatal("expected out-of-range operands to fail")
	}
	if OpAdd.String() != "OP_ADD" {
		t.Fatalf("opcode string mismatch: %s", OpAdd)
	}
	if got := OpCode(255).String(); got != "OpCode(255)" {
		t.Fatalf("unknown opcode string mismatch: %s", got)
	}
}
