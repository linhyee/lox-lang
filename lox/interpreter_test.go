package lox

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	hadError = false
	hadRuntimeError = false
	fn()

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func runCode(code string) (output string, err bool) {
	out := captureStdout(func() {
		scanner := NewScanner(code)
		tokens := scanner.ScanTokens()
		parser := NewParser(tokens)
		stmts := parser.Parse()
		if hadError {
			return
		}
		interp := NewInterpreter()
		resolver := NewResolver(interp)
		resolver.resolve(stmts)
		if hadError {
			return
		}
		interp.Interpret(stmts)
	})
	return out, hadError || hadRuntimeError
}

func normalize(s string) string {
	return strings.TrimRight(strings.Replace(s, "\r\n", "\n", -1), "\n")
}

func assertOutput(t *testing.T, name, code, expected string) {
	t.Helper()
	got, hadErr := runCode(code)
	got = normalize(got)
	want := normalize(expected)
	if hadErr {
		t.Errorf("[%s] unexpected error, output:\n%s", name, got)
		return
	}
	if got != want {
		t.Errorf("[%s] output mismatch\nwant:\n%s\ngot:\n%s", name, want, got)
	}
}

func assertError(t *testing.T, name, code string, contains ...string) {
	t.Helper()
	got, hadErr := runCode(code)
	if !hadErr {
		t.Errorf("[%s] expected error but none occurred, output:\n%s", name, got)
		return
	}
	for _, c := range contains {
		if !strings.Contains(got, c) {
			t.Errorf("[%s] error output expected to contain %q, got:\n%s", name, c, got)
		}
	}
}

// ============================================================
//  字面量 & 基础运算
// ============================================================

func TestLiteralNumbers(t *testing.T) {
	assertOutput(t, "integer literal", `print 42;`, "42")
	assertOutput(t, "float literal", `print 3.14;`, "3.14")
	assertOutput(t, "negative literal", `print -123;`, "-123")
	assertOutput(t, "zero", `print 0;`, "0")
}

func TestLiteralStrings(t *testing.T) {
	assertOutput(t, "simple string", `print "hello";`, "hello")
	assertOutput(t, "empty string", `print "";`, "")
	assertOutput(t, "string concat", `print "a"+"b"+"c";`, "abc")
}

func TestLiteralBoolNil(t *testing.T) {
	assertOutput(t, "true literal", `print true;`, "true")
	assertOutput(t, "false literal", `print false;`, "false")
	assertOutput(t, "nil literal", `print nil;`, "nil")
}

func TestArithmeticOperators(t *testing.T) {
	assertOutput(t, "add", `print 1+2;`, "3")
	assertOutput(t, "sub", `print 10-3;`, "7")
	assertOutput(t, "mul", `print 6*7;`, "42")
	assertOutput(t, "div", `print 10/4;`, "2.5")
	assertOutput(t, "neg", `print -(3+4);`, "-7")
	assertOutput(t, "complex expr", `print 1+2*3-4/2;`, "5")
	assertOutput(t, "paren group", `print (1+2)*(3+4);`, "21")
}

func TestComparisonOperators(t *testing.T) {
	assertOutput(t, "less", `print 1<2;`, "true")
	assertOutput(t, "greater", `print 5>3;`, "true")
	assertOutput(t, "less equal", `print 2<=2; print 2<=3; print 3<=2;`,
		"true\ntrue\nfalse")
	assertOutput(t, "greater equal", `print 5>=5; print 5>=4; print 4>=5;`,
		"true\ntrue\nfalse")
}

func TestEqualityOperators(t *testing.T) {
	assertOutput(t, "eq numbers", `print 1==1; print 1==2;`, "true\nfalse")
	assertOutput(t, "eq strings", `print "ab" == "a"+"b";`, "true")
	assertOutput(t, "neq", `print 1 != 2; print 1 != 1;`, "true\nfalse")
	assertOutput(t, "eq nil", `print nil == nil;`, "true")
	assertOutput(t, "cross type", `print 1 == "1"; print nil == false;`, "false\nfalse")
}

func TestStringConcatNumber(t *testing.T) {
	assertOutput(t, "string concat", `print "hi " + "there";`, "hi there")
	assertError(t, "string + number type error",
		`print "a" + 1;`,
		"operands must be two numbers or two strings")
	assertError(t, "number - string type error",
		`print 1 - "a";`,
		"operands must be numbers")
}

// ============================================================
//  变量 & 赋值 & 作用域
// ============================================================

func TestVarDeclaration(t *testing.T) {
	assertOutput(t, "var with init", `var x = 5; print x;`, "5")
	assertOutput(t, "var no init is nil", `var x; print x;`, "nil")
	assertOutput(t, "var shadowing",
		`var x = 1;
{
  var x = 2;
  print x;
}
print x;`, "2\n1")
}

func TestAssignment(t *testing.T) {
	assertOutput(t, "basic assign", `var x = 1; x = 2; print x;`, "2")
	assertOutput(t, "assign chain via comma", `var a; var b; a = b = 5; print a; print b;`, "5\n5")
	assertOutput(t, "assign in outer scope",
		`var x = 0;
{
  x = 99;
}
print x;`, "99")
}

func TestIncrementDecrement(t *testing.T) {
	assertOutput(t, "prefix ++", `var x = 5; print ++x; print x;`, "6\n6")
	assertOutput(t, "postfix ++", `var x = 5; print x++; print x;`, "5\n6")
	assertOutput(t, "prefix --", `var x = 5; print --x; print x;`, "4\n4")
	assertOutput(t, "postfix --", `var x = 5; print x--; print x;`, "5\n4")
}

func TestBlockScope(t *testing.T) {
	assertOutput(t, "nested block scope",
		`var a = "global a";
var b = "global b";
var c = "global c";
{
  var a = "outer a";
  var b = "outer b";
  {
    var a = "inner a";
    print a;
    print b;
    print c;
  }
  print a;
  print b;
  print c;
}
print a;
print b;
print c;`,
		"inner a\nouter b\nglobal c\n"+
			"outer a\nouter b\nglobal c\n"+
			"global a\nglobal b\nglobal c")
}

func TestErrorUndefined(t *testing.T) {
	assertError(t, "get undefined var", `print noSuchVar;`, "undefined variable")
	assertError(t, "assign to undefined var", `noSuchVar = 5;`, "set undefined variable")
}

// ============================================================
//  控制流
// ============================================================

func TestIfElse(t *testing.T) {
	assertOutput(t, "if then", `if (true) print 1;`, "1")
	assertOutput(t, "if no else", `if (false) print 1; print 2;`, "2")
	assertOutput(t, "if else true", `if (true) print 1; else print 2;`, "1")
	assertOutput(t, "if else false", `if (false) print 1; else print 2;`, "2")
	assertOutput(t, "elif chain",
		`var x = 2;
if (x == 1) print "a";
else if (x == 2) print "b";
else if (x == 3) print "c";
else print "d";`, "b")
	assertOutput(t, "truthy values",
		`if (nil) print 1; else print "nil falsy";
if (0) print "zero truthy";
if ("") print "empty string truthy";`,
		"nil falsy\nzero truthy\nempty string truthy")
}

func TestWhileLoop(t *testing.T) {
	assertOutput(t, "while basic",
		`var i = 0;
while (i < 3) {
  print i;
  i = i + 1;
}`, "0\n1\n2")
	assertOutput(t, "while zero times",
		`var i = 0;
while (i < 0) print i;
print "done";`, "done")
}

func TestForLoop(t *testing.T) {
	assertOutput(t, "for basic",
		`for (var i = 0; i < 3; i = i + 1) print i;`, "0\n1\n2")
	assertOutput(t, "for no init",
		`var i = 0;
for (; i < 3; i = i + 1) print i;`, "0\n1\n2")
	assertOutput(t, "for infinite with break",
		`var i = 0;
for (;;) {
  if (i >= 3) break;
  print i;
  i = i + 1;
}`, "0\n1\n2")
}

func TestBreak(t *testing.T) {
	assertOutput(t, "break while",
		`var i = 0;
while (true) {
  if (i == 5) break;
  print i;
  i = i + 1;
}`, "0\n1\n2\n3\n4")
	assertOutput(t, "break for",
		`var s = 0;
for (var i = 0; i < 100; i = i + 1) {
  s = s + i;
  if (i >= 4) break;
}
print s;`, "10")
	assertOutput(t, "break innermost only",
		`for (var i = 0; i < 3; i = i + 1) {
  for (var j = 0; j < 10; j = j + 1) {
    if (j == 2) break;
    print "i="+string(i)+" j="+string(j);
  }
}`,
		"i=0 j=0\ni=0 j=1\n"+
			"i=1 j=0\ni=1 j=1\n"+
			"i=2 j=0\ni=2 j=1")
}

func TestContinue(t *testing.T) {
	assertOutput(t, "continue while skip even",
		`var i = 0;
while (i < 6) {
  i = i + 1;
  if (i % 2 == 0) continue;
  print i;
}`, "1\n3\n5")
}

func TestNestedControlFlow(t *testing.T) {
	assertOutput(t, "99乘法表简化版",
		`for (var i = 1; i <= 3; i = i + 1) {
  var line = "";
  for (var j = 1; j <= 3; j = j + 1) {
    if (j > i) break;
    line = line + string(i) + "x" + string(j) + " ";
  }
  print line;
}`,
		"1x1 \n2x1 2x2 \n3x1 3x2 3x3 ")
}

// ============================================================
//  逻辑运算 & 三元表达式
// ============================================================

func TestLogicalOperators(t *testing.T) {
	assertOutput(t, "and basic",
		`print true and true; print true and false; print false and true;`,
		"true\nfalse\nfalse")
	assertOutput(t, "or basic",
		`print true or false; print false or true; print false or false;`,
		"true\ntrue\nfalse")
	assertOutput(t, "and short-circuit",
		`var x = 0;
false and (x = 99);
print x;
true and (x = 88);
print x;`, "0\n88")
	assertOutput(t, "or short-circuit",
		`var x = 0;
true or (x = 99);
print x;
false or (x = 77);
print x;`, "0\n77")
	assertOutput(t, "and returns operand",
		`print 1 and 2 and 3;
print nil and 123;
print 0 and "ok";`,
		"3\nnil\n0")
	assertOutput(t, "or returns operand",
		`print nil or 0 or "first";
print false or nil;
print "hi" or 123;`,
		"0\nnil\nhi")
}

func TestTernary(t *testing.T) {
	assertOutput(t, "ternary true branch", `print 1 > 0 ? "yes" : "no";`, "yes")
	assertOutput(t, "ternary false branch", `print 1 < 0 ? "yes" : "no";`, "no")
	assertOutput(t, "ternary only eval taken",
		`var x = 0;
true ? (x = 1) : (x = 99);
print x;
false ? (x = 99) : (x = 2);
print x;`, "1\n2")
}

func TestBang(t *testing.T) {
	assertOutput(t, "bang truthy",
		`print !true; print !false; print !nil; print !0; print !"";`,
		"false\ntrue\ntrue\nfalse\nfalse")
	assertOutput(t, "double bang", `print !!5; print !!nil;`, "true\nfalse")
}

func TestComma(t *testing.T) {
	assertOutput(t, "comma expression",
		`var a = 1;
var b = 2;
print (a = a + 1, b = b + 2, a + b);`, "6")
}

// ============================================================
//  函数 & 闭包
// ============================================================

func TestFunctionDeclarationCall(t *testing.T) {
	assertOutput(t, "fun no args",
		`fun hi() { print "hi"; }
hi();`, "hi")
	assertOutput(t, "fun with args",
		`fun add(a, b) { print a + b; }
add(2, 3);`, "5")
	assertOutput(t, "fun return value",
		`fun add(a, b) { return a + b; }
print add(10, 20) + add(1, 2);`, "33")
	assertOutput(t, "fun implicit return nil",
		`fun nothing() {}
print nothing();`, "nil")
}

func TestReturnInside(t *testing.T) {
	assertOutput(t, "early return",
		`fun abs(x) {
  if (x < 0) return -x;
  return x;
}
print abs(-5);
print abs(7);`, "5\n7")
}

func TestRecursion(t *testing.T) {
	assertOutput(t, "fibonacci",
		`fun fib(n) {
  if (n <= 1) return n;
  return fib(n - 1) + fib(n - 2);
}
print fib(0);
print fib(1);
print fib(6);`, "0\n1\n8")
}

func TestClosure(t *testing.T) {
	assertOutput(t, "closure captures var",
		`fun makeCounter() {
  var i = 0;
  fun count() {
    i = i + 1;
    return i;
  }
  return count;
}
var c = makeCounter();
print c();
print c();
print c();`, "1\n2\n3")
	assertOutput(t, "closure each has own env",
		`fun mulBy(n) {
  fun f(x) { return x * n; }
  return f;
}
var double = mulBy(2);
var triple = mulBy(3);
print double(5);
print triple(5);`, "10\n15")
	assertOutput(t, "shared closure",
		`var f;
var g;
{
  var x = 10;
  fun setX(v) { x = v; }
  fun getX() { return x; }
  f = setX;
  g = getX;
}
f(99);
print g();`, "99")
}

func TestLambda(t *testing.T) {
	assertOutput(t, "lambda literal call",
		`print (fun(a,b){return a*b;})(6,7);`, "42")
	assertOutput(t, "lambda passed to fun",
		`fun applyTwice(f, x) { return f(f(x)); }
print applyTwice(fun(a){return a*a;}, 3);`, "81")
}

func TestWrongArity(t *testing.T) {
	assertError(t, "too few args",
		`fun f(a,b){} f(1);`, "expected 2 arguments but got 1")
	assertError(t, "too many args",
		`fun f(a){} f(1,2);`, "expected 1 arguments but got 2")
}

// ============================================================
//  类 & 继承
// ============================================================

func TestClassDeclareInstance(t *testing.T) {
	assertOutput(t, "class print", `class Foo {} print Foo;`, "Foo")
	assertOutput(t, "instance print", `class Foo {} var f = Foo(); print f;`, "Foo instance")
}

func TestClassFields(t *testing.T) {
	assertOutput(t, "set/get field",
		`class Foo {}
var f = Foo();
f.bar = 123;
f.baz = "hi";
print f.bar;
print f.baz;`, "123\nhi")
}

func TestClassMethods(t *testing.T) {
	assertOutput(t, "basic method call",
		`class Math {
  square(x) { return x * x; }
}
var m = Math();
print m.square(5);`, "25")
	assertOutput(t, "this inside method",
		`class Person {
  setName(n) { this.name = n; }
  greet() { return "I am " + this.name; }
}
var p = Person();
p.setName("Alice");
print p.greet();`, "I am Alice")
}

func TestClassInitializer(t *testing.T) {
	assertOutput(t, "init auto returns this",
		`class Pair {
  init(a, b) {
    this.a = a;
    this.b = b;
  }
}
var p = Pair(11, 22);
print p.a;
print p.b;`, "11\n22")
}

func TestInheritance(t *testing.T) {
	assertOutput(t, "inherit method",
		`class A {
  f() { return "A.f"; }
}
class B < A {}
var b = B();
print b.f();`, "A.f")
	assertOutput(t, "override method",
		`class A {
  f() { return "A.f"; }
}
class B < A {
  f() { return "B.f"; }
}
var b = B();
print b.f();`, "B.f")
	assertOutput(t, "super call",
		`class A {
  say() { return "A"; }
}
class B < A {
  say() { return super.say() + "+B"; }
}
class C < B {
  say() { return super.say() + "+C"; }
}
var c = C();
print c.say();`, "A+B+C")
}

func TestInitializerInheritance(t *testing.T) {
	assertOutput(t, "superclass init called implicitly via class call arg count",
		`class Base {
  init(x) { this.x = x; }
  getX() { return this.x; }
}
class Derived < Base {}
var d = Derived(42);
print d.getX();`, "42")
}

// ============================================================
//  数组 & 内置函数
// ============================================================

func TestArrayLiteral(t *testing.T) {
	assertOutput(t, "empty array", `print [];`, "[]")
	assertOutput(t, "homogenous", `print [1,2,3];`, "[1,2,3]")
	assertOutput(t, "heterogeneous", `print [1,"two",true,nil];`, "[1,two,true,nil]")
	assertOutput(t, "nested array",
		`var m = [[1,2],[3,4]];
print m[0][1];
print m[1][0];`, "2\n3")
}

func TestArrayIndex(t *testing.T) {
	assertOutput(t, "index read",
		`var a = [10,20,30,40,50];
print a[0];
print a[2];
print a[4];`, "10\n30\n50")
}

func TestArraySet(t *testing.T) {
	assertOutput(t, "index write",
		`var a = [1,2,3];
a[0] = 99;
a[2] = a[2] * 10;
print a;`, "[99,2,30]")
	assertOutput(t, "array add (append)",
		`var a = [1];
a[] = 2;
a[] = 3;
print a;`, "[1,2,3]")
}

func TestBuiltinLen(t *testing.T) {
	assertOutput(t, "len array", `print len([1,2,3]); print len([]);`, "3\n0")
	assertOutput(t, "len string",
		`print len("");
print len("hi");
print len("你好");`, "0\n2\n2")
}

func TestBuiltinString(t *testing.T) {
	assertOutput(t, "string from number",
		`print string(42); print string(3.14); print string(-1);`, "42\n3.14\n-1")
	assertOutput(t, "string from bool/nil",
		`print string(true); print string(false); print string(nil);`,
		"true\nfalse\n<nil>")
	assertOutput(t, "string concat via conversion",
		`var x = 7;
print "x is " + string(x);`, "x is 7")
}

func TestBuiltinClock(t *testing.T) {
	out, hadErr := runCode(`var t1 = clock();
var s = 0;
for (var i = 0; i < 10000; i = i + 1) s = s + i;
var t2 = clock();
print t2 >= t1;`)
	if hadErr {
		t.Errorf("clock builtin caused error: %s", out)
	}
	if normalize(out) != "true" {
		t.Errorf("clock expected t2>=t1, got:\n%s", out)
	}
}

func TestArrayOutOfBounds(t *testing.T) {
	assertError(t, "read OOB", `var a = [1]; print a[99];`, "out of bound")
	assertError(t, "write OOB", `var a = [1]; a[99] = 5;`, "out of bound")
}

// ============================================================
//  综合测试
// ============================================================

func TestBubbleSortIntegration(t *testing.T) {
	assertOutput(t, "bubble sort",
		`fun sort(list) {
  var n = len(list);
  for (var i = 0; i < n; i = i + 1) {
    for (var j = 0; j < n - i - 1; j = j + 1) {
      if (list[j] > list[j + 1]) {
        var t = list[j];
        list[j] = list[j + 1];
        list[j + 1] = t;
      }
    }
  }
}
var a = [5, 1, 4, 2, 8, 0, 3];
sort(a);
print a;`, "[0,1,2,3,4,5,8]")
}

func TestFactorialIntegration(t *testing.T) {
	assertOutput(t, "factorial",
		`fun fact(n) {
  if (n <= 1) return 1;
  return n * fact(n - 1);
}
print fact(0);
print fact(1);
print fact(5);
print fact(10);`, "1\n1\n120\n3628800")
}
