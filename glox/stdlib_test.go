package glox

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func runStdlibScript(t *testing.T, source string, opts Options) (string, string, error) {
	t.Helper()
	var out bytes.Buffer
	var errOut bytes.Buffer
	opts.Stdout = &out
	opts.Stderr = &errOut
	vm := NewVM(opts)
	err := vm.RunString(source)
	return out.String(), errOut.String(), err
}

func TestStdlibCoreModules(t *testing.T) {
	out, stderr, err := runStdlibScript(t, `
var math = import("math");
var mathAgain = import("math.lox");
var mathViaStdlib = import("stdlib/math");
var string = import("string");
var json = import("json");
var std = import("std");
var times = import("times");
var regexp = import("regexp");

print math == mathAgain;
print mathAgain == mathViaStdlib;
print math.abs(-7);
print math.mod(10, 4);
print math.pow(2, 3);
print math.sqrt(9);
print math.clamp(12, 1, 10);
print math.min(3, 1, 2);
print math.max(3, 1, 2);
print math.isInf(1.0 / 0.0);

print string.upper("ab");
print string.lower("AB");
print string.len("你a");
print string.slice("abcdef", 1, 4);
print string.split("a,b,c", ",")[1];
print string.join(["x", 2, true], "-");
print string.index("你好吗", "好");

var data = json.decode("{\"a\":1,\"b\":[true,2.5]}");
print data.a;
print data.b[1];
print json.encode({b: true, a: 1});
print json.valid("{\"x\":1}");
print json.compact(" { \"x\" : 1 } ");
print string.contains(json.indent("{\"x\":1}", "", "  "), "\n  \"x\"");

print regexp.match("^[a-z]+$", "abc");
print regexp.find("[0-9]+", "a12b");
print regexp.findSubmatch("([a-z]+)([0-9]+)", "ab12")[2];
print regexp.replace("[0-9]+", "a12b", "#");
print regexp.split(",", "a,b,c", -1)[2];

var epoch = times.parse(times.RFC3339, "1970-01-01T00:00:00Z");
print epoch;
print times.format(epoch, times.RFC3339);
print times.add(epoch, 60);
print times.date(1970, 1, 1, 0, 1, 0);

print std.typeOf(math);
print std.isNumber(math.PI);
print std.isString(string.lower("A"));
print std.has(math, "PI");
print std.keys({b: 2, a: 1})[0];
print std.values({b: 2, a: 1})[0];
print std.range(1, 5, 2)[1];
print std.number("1e2");
print std.bool(nil);
`, Options{})
	if err != nil {
		t.Fatalf("stdlib script failed: %v\nstderr:\n%s", err, stderr)
	}

	want := "true\ntrue\n" +
		"7\n2\n8\n3\n10\n1\n3\ntrue\n" +
		"AB\nab\n2\nbcd\nb\nx-2-true\n1\n" +
		"1\n2.5\n{\"a\":1,\"b\":true}\ntrue\n{\"x\":1}\ntrue\n" +
		"true\n12\n12\na#b\nc\n" +
		"0\n1970-01-01T00:00:00Z\n60\n60\n" +
		"module\ntrue\ntrue\ntrue\na\n1\n3\n100\nfalse\n"
	if out != want {
		t.Fatalf("unexpected output\nwant:\n%q\ngot:\n%q", want, out)
	}
}

func TestStdlibOSModule(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "nested")
	file := filepath.Join(nested, "note.txt")
	cleanJoined := filepath.Clean(filepath.Join(dir, ".", "nested", "..", "nested", "note.txt"))

	source := fmt.Sprintf(`
var os = import("os");
var string = import("string");

print os.args()[0];
print os.args()[1];
print os.cwd() == "";
print os.exists(%q);
print os.mkdirAll(%q);
print os.isDir(%q);
print os.writeFile(%q, "hello");
print os.exists(%q);
print os.isFile(%q);
print os.readFile(%q);
print os.base(%q);
print os.ext(%q);
print os.dir(%q) == %q;
print os.clean(os.join(%q, ".", "nested", "..", "nested", "note.txt")) == %q;
print string.len(os.tempDir()) > 0;
print os.setenv("GLOX_STDLIB_TEST", "ok");
print os.getenv("GLOX_STDLIB_TEST");
print os.unsetenv("GLOX_STDLIB_TEST");
print os.getenv("GLOX_STDLIB_TEST");
print os.remove(%q);
print os.exists(%q);
`, file, nested, nested, file, file, file, file, file, file, file, filepath.Dir(file), dir, cleanJoined, file, file)

	out, stderr, err := runStdlibScript(t, source, Options{Args: []string{"one", "two"}})
	if err != nil {
		t.Fatalf("os stdlib script failed: %v\nstderr:\n%s", err, stderr)
	}

	want := "one\ntwo\nfalse\nfalse\ntrue\ntrue\ntrue\ntrue\ntrue\nhello\n" +
		"note.txt\n.txt\ntrue\ntrue\ntrue\ntrue\nok\ntrue\n\ntrue\nfalse\n"
	if out != want {
		t.Fatalf("unexpected output\nwant:\n%q\ngot:\n%q", want, out)
	}
}

func TestStdlibDoesNotRequireLoxWrapperFiles(t *testing.T) {
	for _, name := range []string{"os", "string", "json", "math", "std", "times", "regexp"} {
		if _, err := os.Stat(filepath.Join("stdlib", name+".lox")); !os.IsNotExist(err) {
			t.Fatalf("stdlib module %s should be native, wrapper stat err=%v", name, err)
		}
	}
}
