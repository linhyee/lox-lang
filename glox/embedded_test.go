package glox

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestNumericTypesAndArithmetic(t *testing.T) {
	var out bytes.Buffer
	vm := NewVM(Options{Stdout: &out})
	result, err := vm.DoString(`
var i = 1 + 2 * 3;
var q = 5 / 2;
var f = 1 + 2.5;
var sci = 1e3 + 5;
print i;
print q;
print f;
print sci;
i;
`)
	if err != nil {
		t.Fatalf("DoString failed: %v", err)
	}
	if result != int64(7) {
		t.Fatalf("last result should be int64(7), got %#v (%T)", result, result)
	}
	if out.String() != "7\n2\n3.5\n1005\n" {
		t.Fatalf("unexpected output: %q", out.String())
	}

	i, ok := vm.GetGlobal("i")
	if !ok || i != int64(7) {
		t.Fatalf("global i mismatch: %#v ok=%v", i, ok)
	}
	q, ok := vm.GetGlobal("q")
	if !ok || q != int64(2) {
		t.Fatalf("global q mismatch: %#v ok=%v", q, ok)
	}
	f, ok := vm.GetGlobal("f")
	if !ok || f != float64(3.5) {
		t.Fatalf("global f mismatch: %#v ok=%v", f, ok)
	}
	sci, ok := vm.GetGlobal("sci")
	if !ok || sci != float64(1005) {
		t.Fatalf("global sci mismatch: %#v ok=%v", sci, ok)
	}
}

func TestEmbeddedScriptingAPI(t *testing.T) {
	vm := NewVM(Options{})
	vm.SetGlobal("host", int64(10))

	result, err := vm.DoString(`var a = 3; a = a + host`)
	if err != nil {
		t.Fatalf("DoString failed: %v", err)
	}
	if result != int64(13) {
		t.Fatalf("DoString result mismatch: %#v (%T)", result, result)
	}
	if value, ok := vm.GetGlobal("a"); !ok || value != int64(13) {
		t.Fatalf("GetGlobal(a) mismatch: %#v ok=%v", value, ok)
	}
	if value, ok := vm.GetGlobal("host"); !ok || value != int64(10) {
		t.Fatalf("GetGlobal(host) mismatch: %#v ok=%v", value, ok)
	}

	result, err = vm.DoString(`a = a + 0.5`)
	if err != nil {
		t.Fatalf("second DoString failed: %v", err)
	}
	if result != float64(13.5) {
		t.Fatalf("mixed numeric result mismatch: %#v (%T)", result, result)
	}
	if value, ok := vm.GetGlobal("a"); !ok || value != float64(13.5) {
		t.Fatalf("GetGlobal(a) after float assignment mismatch: %#v ok=%v", value, ok)
	}
}

func TestEmbeddedDoFileResultAndGlobals(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.lox")
	if err := os.WriteFile(path, []byte(`var fileValue = 40; fileValue = fileValue + 2`), 0o644); err != nil {
		t.Fatal(err)
	}

	vm := NewVM(Options{RootDir: dir})
	result, err := vm.DoFile(path)
	if err != nil {
		t.Fatalf("DoFile failed: %v", err)
	}
	if result != int64(42) {
		t.Fatalf("DoFile result mismatch: %#v (%T)", result, result)
	}
	if value, ok := vm.GetGlobal("fileValue"); !ok || value != int64(42) {
		t.Fatalf("GetGlobal(fileValue) mismatch: %#v ok=%v", value, ok)
	}
}
