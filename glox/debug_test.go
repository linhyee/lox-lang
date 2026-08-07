package glox

import (
	"bytes"
	"strings"
	"testing"
)

func TestDisassemblerFormatsInstructions(t *testing.T) {
	var chunk Chunk
	constant := chunk.AddConstant("name")
	fn := &Function{Name: "inner", UpvalueCount: 1}
	fnConst := chunk.AddConstant(fn)

	write := func(op OpCode, line int, operands ...byte) {
		chunk.Write(byte(op), line)
		for _, operand := range operands {
			chunk.Write(operand, line)
		}
	}
	write(OpConstant, 1, byte(constant))
	write(OpJump, 1, 0, 2)
	write(OpInvoke, 2, byte(constant), 3)
	write(OpClosure, 3, byte(fnConst), 1, 2, 1)
	write(OpReturn, 4)

	var out bytes.Buffer
	dis := NewDisassembler(&out)
	dis.Chunk(&chunk, "test")
	got := out.String()
	for _, want := range []string{
		"== test ==",
		"0000    1 OP_CONSTANT",
		"'name'",
		"OP_JUMP",
		"->",
		"OP_INVOKE",
		"(3 args)",
		"OP_CLOSURE",
		"<fn inner>",
		"local",
		"const",
		"OP_RETURN",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("disassembly missing %q:\n%s", want, got)
		}
	}

	out.Reset()
	next := dis.Instruction(&chunk, 999)
	if next != len(chunk.Code) || !strings.Contains(out.String(), "<invalid offset>") {
		t.Fatalf("invalid offset disassembly mismatch next=%d out=%q", next, out.String())
	}

	out.Reset()
	dis.Stack([]Value{float64(1), "x", nil})
	if got := out.String(); got != "          [ 1 ][ x ][ nil ]\n" {
		t.Fatalf("stack trace mismatch: %q", got)
	}
}

func TestVMDisassembleAndTraceIntegration(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var debug bytes.Buffer
	vm := NewVM(Options{
		Stdout:              &stdout,
		Stderr:              &stderr,
		DebugWriter:         &debug,
		DebugDisassemble:    true,
		DebugTraceExecution: true,
	})
	if err := vm.RunString(`var a = 1; print a + 2;`); err != nil {
		t.Fatalf("run failed: %v stderr=%s", err, stderr.String())
	}
	if stdout.String() != "3\n" {
		t.Fatalf("stdout mismatch: %q", stdout.String())
	}
	got := debug.String()
	for _, want := range []string{
		"== script ==",
		"OP_DEFINE_GLOBAL",
		"OP_PRINT",
		"[ <script> ]",
		"OP_CONSTANT",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("debug output missing %q:\n%s", want, got)
		}
	}
}
