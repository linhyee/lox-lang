package glox

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runSyntaxScript(t *testing.T, source string, opts Options) (string, string, error) {
	t.Helper()
	var out bytes.Buffer
	var errOut bytes.Buffer
	opts.Stdout = &out
	opts.Stderr = &errOut
	vm := NewVM(opts)
	err := vm.RunString(source)
	return out.String(), errOut.String(), err
}

func assertScriptOutput(t *testing.T, name, source, want string) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		t.Helper()
		out, stderr, err := runSyntaxScript(t, source, Options{})
		if err != nil {
			t.Fatalf("run failed: %v\nstderr:\n%s", err, stderr)
		}
		if out != want {
			t.Fatalf("unexpected output\nwant:\n%q\ngot:\n%q", want, out)
		}
	})
}

func assertScriptError(t *testing.T, name, source, want string) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		t.Helper()
		_, stderr, err := runSyntaxScript(t, source, Options{})
		if err == nil {
			t.Fatalf("expected error containing %q", want)
		}
		if !strings.Contains(err.Error(), want) && !strings.Contains(stderr, want) {
			t.Fatalf("expected error containing %q\nerr: %v\nstderr:\n%s", want, err, stderr)
		}
	})
}

func TestSyntaxLiteralsOperatorsAndExpressions(t *testing.T) {
	assertScriptOutput(t, "literals", `
print nil;
print true;
print false;
print 123;
print "a\nb";
`, "nil\ntrue\nfalse\n123\na\nb\n")

	assertScriptOutput(t, "arithmetic precedence grouping unary", `
print 1 + 2 * 3;
print (1 + 2) * 3;
print -3 + 10 / 2;
print !false;
print !nil;
print !true;
`, "7\n9\n2\ntrue\ntrue\nfalse\n")

	assertScriptOutput(t, "comparison equality", `
print 3 > 2;
print 3 >= 3;
print 2 < 3;
print 2 <= 2;
print 1 == 1;
print 1 != 2;
print "a" == "a";
`, "true\ntrue\ntrue\ntrue\ntrue\ntrue\ntrue\n")

	assertScriptOutput(t, "logical short circuit ternary", `
fun yes() { print "yes"; return true; }
fun no() { print "no"; return false; }
print false and yes();
print true or yes();
print true ? "then" : "else";
print false ? "then" : "else";
print no() and yes();
print yes() or no();
`, "false\ntrue\nthen\nelse\nno\nfalse\nyes\ntrue\n")

	assertScriptOutput(t, "assignment and increments", `
var a = 1;
print a++;
print a;
print ++a;
print a--;
print a;
print --a;
`, "1\n2\n3\n3\n2\n1\n")
}

func TestSyntaxDeclarationsScopesAndControlFlow(t *testing.T) {
	assertScriptOutput(t, "var const and block shadowing", `
var a, b = 2, c = 3;
print a;
print b + c;
const k = "global";
{
  var k = "local";
  const c = "const local";
  print k;
  print c;
}
print k;
`, "nil\n5\nlocal\nconst local\nglobal\n")

	assertScriptOutput(t, "if else while break continue", `
var total = 0;
var i = 0;
while (i < 6) {
  i = i + 1;
  if (i == 2) continue;
  if (i == 5) break;
  total = total + i;
}
if (total == 8) print "ok"; else print "bad";
`, "ok\n")

	assertScriptOutput(t, "for clauses", `
var out = "";
for (var i = 0; i < 4; i = i + 1) {
  out = out + string(i);
}
print out;
for (;;) {
  print "once";
  break;
}
`, "0123\nonce\n")

	assertScriptOutput(t, "for const initializer", `
for (const start = 1; start < 2; ) {
  print start;
  break;
}
`, "1\n")
}

func TestSyntaxFunctionsClosuresAndErrors(t *testing.T) {
	assertScriptOutput(t, "functions recursion lambda closure", `
fun fib(n) {
  if (n <= 1) return n;
  return fib(n - 1) + fib(n - 2);
}
print fib(6);
fun makeAdder(a) {
  return fun(b) {
    return a + b;
  };
}
var add10 = makeAdder(10);
print add10(7);
fun makeCounter() {
  var count = 0;
  fun inc() {
    count = count + 1;
    return count;
  }
  return inc;
}
var c = makeCounter();
print c();
print c();
`, "8\n17\n1\n2\n")

	assertScriptOutput(t, "nested upvalue closes over block local", `
fun outer() {
  var f;
  {
    var x = "closed";
    f = fun() { return x; };
  }
  return f;
}
print outer()();
`, "closed\n")

	assertScriptError(t, "return at top level", `return 1;`, "can't return from top-level code")
	assertScriptError(t, "local const assignment", `{ const x = 1; x = 2; }`, "cannot assign to const 'x'")
	assertScriptError(t, "global const assignment", `const x = 1; x = 2;`, "cannot assign to const 'x'")
	assertScriptError(t, "const without initializer", `const x;`, "const declaration requires an initializer")
}

func TestSyntaxClassesListsAndMaps(t *testing.T) {
	assertScriptOutput(t, "classes this fields methods inheritance super", `
class Doughnut {
  init(flavor) {
    this.flavor = flavor;
  }
  describe() {
    return "doughnut:" + this.flavor;
  }
}
class Boston < Doughnut {
  describe() {
    return super.describe() + ":cream";
  }
  parentMethod() {
    return super.describe;
  }
}
var b = Boston("vanilla");
print b.flavor;
b.flavor = "chocolate";
print b.describe();
var method = b.describe;
print method();
print b.parentMethod()();
`, "vanilla\ndoughnut:chocolate:cream\ndoughnut:chocolate:cream\ndoughnut:chocolate\n")

	assertScriptOutput(t, "lists", `
var xs = [];
xs.push(1);
xs.push(2);
xs.insertAt(1, 9);
print xs[0];
print xs[1];
print xs[2];
print xs.size();
print xs.remove(1);
print xs.pop();
print len(xs);
xs[0] = 7;
print xs[0];
`, "1\n9\n2\n3\n9\n2\n1\n7\n")

	assertScriptOutput(t, "maps", `
var key = "dyn";
var m = {a: 1, "b": 2, [key]: 3};
print m.a;
print m["b"];
print m["dyn"];
m.c = 4;
m["d"] = 5;
print m.c + m["d"];
print m.has("a");
print m.remove("a");
print m.has("a");
print m.size();
print len(m);
`, "1\n2\n3\n9\ntrue\n1\nfalse\n4\n4\n")
}

func TestSyntaxImportExportAndFFI(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("hello.lox", `
export const seed = 10;
export var counter = 0;
export fun bump(n) {
  counter = counter + n;
  return counter + seed;
}
export class A {
  init(v) { this.v = v; }
  value() { return this.v; }
}
var c;
class C { value() { return "C"; } }
fun ok() { return "ok"; }
export { c, C, ok }
c = "late";
`)
	write("something/a.lox", `export var x = 5;`)
	write("math/b.lox", `export fun twice(n) { return n * 2; }`)

	var out bytes.Buffer
	var stderr bytes.Buffer
	vm := NewVM(Options{Stdout: &out, Stderr: &stderr, RootDir: dir})
	vm.DefineNative("hostJoin", 2, func(vm *VM, args []Value) (Value, error) {
		return Stringify(args[0]) + ":" + Stringify(args[1]), nil
	})
	vm.RegisterNativeModule("host", map[string]Value{
		"name": "native",
		"add": &NativeFunction{Name: "host.add", Arity: 2, Fn: func(vm *VM, args []Value) (Value, error) {
			left, _ := AsFloat64(args[0])
			right, _ := AsFloat64(args[1])
			return left + right, nil
		}},
	})

	err := vm.RunString(`
var a = import("hello.lox");
var again = import("./hello.lox");
var b = import("./something/a.lox");
var c = import("/math/b.lox");
var host = import("host");
print a.bump(1);
print again.bump(2);
print b.x;
print c.twice(6);
print a.A(42).value();
print a.c;
print a.C().value();
print a.ok();
print host.name;
print host.add(3, 4);
print hostJoin("x", 9);
`)
	if err != nil {
		t.Fatalf("run failed: %v\nstderr:\n%s", err, stderr.String())
	}
	want := "11\n13\n5\n12\n42\nlate\nC\nok\nnative\n7\nx:9\n"
	if out.String() != want {
		t.Fatalf("unexpected output\nwant:\n%q\ngot:\n%q", want, out.String())
	}
}

func TestSyntaxModuleErrors(t *testing.T) {
	t.Run("non exported property is hidden", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "m.lox"), []byte(`var hidden = 1; export var shown = 2;`), 0o644); err != nil {
			t.Fatal(err)
		}
		_, stderr, err := runSyntaxScript(t, `var m = import("m.lox"); print m.hidden;`, Options{RootDir: dir})
		if err == nil {
			t.Fatal("expected hidden export error")
		}
		if !strings.Contains(err.Error(), "undefined export 'hidden'") && !strings.Contains(stderr, "undefined export 'hidden'") {
			t.Fatalf("unexpected error: %v\nstderr:\n%s", err, stderr)
		}
	})

	t.Run("circular import", func(t *testing.T) {
		dir := t.TempDir()
		files := map[string]string{
			"a.lox": `var b = import("b.lox"); export var a = 1;`,
			"b.lox": `var c = import("c.lox"); export var b = 1;`,
			"c.lox": `var a = import("a.lox"); export var c = 1;`,
		}
		for name, body := range files {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		_, stderr, err := runSyntaxScript(t, `var a = import("a.lox");`, Options{RootDir: dir})
		if err == nil {
			t.Fatal("expected circular import error")
		}
		text := strings.ToLower(fmt.Sprint(err) + "\n" + stderr)
		if !strings.Contains(text, "circular import") {
			t.Fatalf("unexpected error: %v\nstderr:\n%s", err, stderr)
		}
	})
}
