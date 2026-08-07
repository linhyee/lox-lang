package glox

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runScript(t *testing.T, source string, opts Options) (string, error) {
	t.Helper()
	var out bytes.Buffer
	var errOut bytes.Buffer
	opts.Stdout = &out
	opts.Stderr = &errOut
	vm := NewVM(opts)
	err := vm.RunString(source)
	if err != nil && errOut.Len() > 0 {
		return out.String(), err
	}
	return out.String(), err
}

func TestCoreBytecodeVM(t *testing.T) {
	out, err := runScript(t, `
var a = 1;
const b = 2;
print a + b;
fun makeCounter() {
  var count = 0;
  return fun() {
    count = count + 1;
    return count;
  };
}
var c = makeCounter();
print c();
print c();
class Doughnut {
  init(v) { this.v = v; }
  cook() { return "cook " + this.v; }
}
class Boston < Doughnut {
  cook() { return super.cook() + "!"; }
}
print Boston("cream").cook();
var xs = [1, 2];
xs.push(3);
print xs[2];
var m = {a: 1, "b": 2};
m["c"] = 3;
print m.a + m["b"] + m["c"];
`, Options{})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	want := "3\n1\n2\ncook cream!\n3\n6\n"
	if out != want {
		t.Fatalf("unexpected output\nwant:\n%q\ngot:\n%q", want, out)
	}
}

func TestConstScopeAndClosure(t *testing.T) {
	_, err := runScript(t, `
fun outer() {
  const x = 1;
  fun inner() { x = 2; }
  return inner;
}
`, Options{})
	if err == nil || !strings.Contains(err.Error(), "cannot assign to const 'x'") {
		t.Fatalf("expected const closure compile error, got %v", err)
	}
}

func TestImportExportCacheAndFFI(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("mod.lox", `
export var counter = 0;
export fun bump() {
  counter = counter + 1;
  return counter;
}
export class Box {
  init(v) { this.v = v; }
  value() { return this.v; }
}
var hidden = 99;
var late;
fun ok() { return "ok"; }
export { late, ok }
late = "late";
`)
	var out bytes.Buffer
	var errOut bytes.Buffer
	vm := NewVM(Options{Stdout: &out, Stderr: &errOut, RootDir: dir})
	vm.DefineNative("hostAdd", 2, func(vm *VM, args []Value) (Value, error) {
		left, _ := AsFloat64(args[0])
		right, _ := AsFloat64(args[1])
		return left + right, nil
	})
	err := vm.RunString(`
var a = import("mod.lox");
var b = import("./mod.lox");
print a.bump();
print b.bump();
print a.Box(7).value();
print a.late;
print a.ok();
print hostAdd(4, 5);
`)
	if err != nil {
		t.Fatalf("run failed: %v stderr=%s", err, errOut.String())
	}
	want := "1\n2\n7\nlate\nok\n9\n"
	if out.String() != want {
		t.Fatalf("unexpected output\nwant:\n%q\ngot:\n%q", want, out.String())
	}
}

func TestCircularImport(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.lox"), []byte(`var b = import("b.lox"); export var a = 1;`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.lox"), []byte(`var a = import("a.lox"); export var b = 1;`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := runScript(t, `var a = import("a.lox");`, Options{RootDir: dir})
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "circular import") {
		t.Fatalf("expected circular import error, got %v", err)
	}
}

func TestReplKeepsStateAndClearsDiagnostics(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	vm := NewVM(Options{Stdout: &out, Stderr: &errOut})

	if err := vm.RunString(`2+2;`); err != nil {
		t.Fatalf("expression statement failed: %v", err)
	}
	if err := vm.RunString(`print 2+2;`); err != nil {
		t.Fatalf("print failed: %v", err)
	}
	if err := vm.RunString(`var a = 4;`); err != nil {
		t.Fatalf("var declaration failed: %v", err)
	}
	if err := vm.RunString(`print a;`); err != nil {
		t.Fatalf("repl variable did not persist: %v\nstderr:\n%s", err, errOut.String())
	}
	if out.String() != "4\n4\n" {
		t.Fatalf("unexpected output: %q", out.String())
	}

	errOut.Reset()
	if err := vm.RunString(`missing;`); err == nil {
		t.Fatal("expected undefined variable error")
	}
	if got := errOut.String(); strings.Count(got, "undefined variable 'missing'") != 1 {
		t.Fatalf("expected one runtime diagnostic, got:\n%s", got)
	}

	errOut.Reset()
	if result, err := vm.DoString(`a`); err != nil || result != int64(4) {
		t.Fatalf("valid final expression without semicolon failed: result=%#v err=%v", result, err)
	}
	if errOut.Len() != 0 {
		t.Fatalf("expected no diagnostic for final expression, got:\n%s", errOut.String())
	}

	errOut.Reset()
	if err := vm.RunString(`var`); err == nil {
		t.Fatal("expected compile error")
	}
	if got := errOut.String(); !strings.Contains(got, "expect variable name") ||
		strings.Contains(got, "undefined variable 'missing'") {
		t.Fatalf("stale diagnostic leaked into compile error:\n%s", got)
	}

	errOut.Reset()
	if err := vm.RunString(`a;`); err != nil {
		t.Fatalf("valid repl expression after errors failed: %v\nstderr:\n%s", err, errOut.String())
	}
	if errOut.Len() != 0 {
		t.Fatalf("expected no stale diagnostics, got:\n%s", errOut.String())
	}
}
