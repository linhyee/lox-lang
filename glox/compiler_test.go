package glox

import (
	"bytes"
	"strings"
	"testing"
)

func compileForTest(t *testing.T, source string) *Function {
	t.Helper()
	var errOut bytes.Buffer
	diag := NewDiagnostics(&errOut)
	function, err := Compile(source, "test.lox", NewModule("test.lox"), diag)
	if err != nil {
		t.Fatalf("compile failed: %v\nstderr:\n%s\nsource:\n%s", err, errOut.String(), source)
	}
	return function
}

func collectOpcodes(function *Function, out map[OpCode]int) {
	if function == nil {
		return
	}
	for offset := 0; offset < len(function.Chunk.Code); {
		op := OpCode(function.Chunk.Code[offset])
		out[op]++
		switch instructionTable[op].Operand {
		case OperandByte, OperandConstant:
			offset += 2
		case OperandJump, OperandLoop, OperandInvoke:
			if instructionTable[op].Operand == OperandInvoke {
				offset += 3
			} else {
				offset += 3
			}
		case OperandClosure:
			offset += 2
			fn, _ := function.Chunk.Constant(int(function.Chunk.Code[offset-1])).(*Function)
			if fn != nil {
				offset += fn.UpvalueCount * 3
			}
		default:
			offset++
		}
	}
	for _, constant := range function.Chunk.Constants {
		if child, ok := constant.(*Function); ok {
			collectOpcodes(child, out)
		}
	}
}

func TestCompilerGeneratesAllOpcodeShapes(t *testing.T) {
	sources := []string{
		`
print nil; print true; print false;
const k = 1;
var g = 1;
g = g + k - 2 * 3 / 4;
g++;
print g > 0;
print g >= 0;
print g < 10;
print g <= 10;
print g == 1;
print g != 2;
print !false;
print -g;
`,
		`
fun localOps(a) {
  var x = a;
  x = x + 1;
  return x;
}
print localOps(1);
`,
		`
fun outer() {
  var x = 0;
  fun inner() {
    x = x + 1;
    return x;
  }
  return inner;
}
var c = outer();
print c();
`,
		`
fun closeBlock() {
  var f;
  {
    var x = "closed";
    f = fun() { return x; };
  }
  return f;
}
print closeBlock()();
`,
		`
class Base {
  init(v) { this.v = v; }
  value() { return this.v; }
  method() { return "base"; }
}
class Child < Base {
  method() { return super.method() + "!"; }
  parent() { return super.method; }
}
var obj = Child("x");
obj.v = "y";
print obj.value();
print obj.method();
print obj.parent()();
`,
		`
var xs = [1, 2, 3];
xs[0] = 9;
print xs[0];
var key = "dyn";
var m = {a: 1, "b": 2, [key]: 3};
m.c = 4;
m["d"] = 5;
print m.a + m["b"] + m["dyn"] + m.c + m["d"];
`,
		`
var i = 0;
while (i < 2) {
  i = i + 1;
  if (i == 1) continue;
  break;
}
for (var j = 0; j < 2; j = j + 1) print j;
`,
	}

	got := map[OpCode]int{}
	for _, source := range sources {
		collectOpcodes(compileForTest(t, source), got)
	}

	want := []OpCode{
		OpConstant, OpNil, OpTrue, OpFalse, OpPop, OpDup,
		OpGetLocal, OpSetLocal, OpGetGlobal, OpDefineGlobal, OpDefineGlobalConst, OpSetGlobal,
		OpGetUpvalue, OpSetUpvalue, OpGetProperty, OpSetProperty, OpGetSuper,
		OpGetIndex, OpSetIndex, OpList, OpMap, OpMapSet,
		OpEqual, OpGreater, OpLess, OpAdd, OpSubtract, OpMultiply, OpDivide, OpNot, OpNegate,
		OpPrint, OpJump, OpJumpIfFalse, OpLoop, OpCall, OpInvoke, OpSuperInvoke, OpClosure,
		OpCloseUpvalue, OpReturn, OpClass, OpInherit, OpMethod,
		OpPopResult,
	}
	for _, op := range want {
		if got[op] == 0 {
			t.Fatalf("opcode %s was not generated; generated=%v", op, got)
		}
	}
}

func TestCompilerErrors(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{"missing semicolon", `print 1`, "expect ';' after value"},
		{"duplicate local", `{ var a = 1; var a = 2; }`, "already a variable with this name"},
		{"read own initializer", `{ var a = a; }`, "can't read local variable in its own initializer"},
		{"return top level", `return 1;`, "can't return from top-level code"},
		{"break outside loop", `break;`, "can't use 'break' outside of a loop"},
		{"continue outside loop", `continue;`, "can't use 'continue' outside of a loop"},
		{"this outside class", `print this;`, "can't use 'this' outside of a class"},
		{"super outside class", `super.foo();`, "can't use 'super' outside of a class"},
		{"class inherits itself", `class A < A {}`, "a class can't inherit from itself"},
		{"const no initializer", `const x;`, "const declaration requires an initializer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var errOut bytes.Buffer
			_, err := Compile(tt.source, "bad.lox", NewModule("bad.lox"), NewDiagnostics(&errOut))
			if err == nil {
				t.Fatal("expected compile error")
			}
			if !strings.Contains(err.Error(), tt.want) && !strings.Contains(errOut.String(), tt.want) {
				t.Fatalf("expected %q in error\nerr=%v\nstderr=%s", tt.want, err, errOut.String())
			}
		})
	}
}
