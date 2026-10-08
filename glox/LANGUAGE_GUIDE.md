# Glox Language Guide

Glox is a bytecode VM implementation of Lox written in Go. It follows the spirit of clox, and adds const bindings, modules, imports, lists, maps, native modules, an embedding API, and a built-in standard library.

This guide describes the language implemented by the `glox` package in this directory.

## Running Code

From the `glox` directory:

```powershell
go run ./cmd/glox
go run ./cmd/glox path/to/script.lox arg1 arg2
```

After building the command, the executable has the same shape:

```text
glox [script] [args...]
```

Without a script, Glox starts a REPL. REPL global variables persist between input lines:

```lox
> var a = 4;
> print a;
4
> a
```

Expression statements evaluate silently unless you use `print`, but the embedding API can read the last expression result.

## Lexical Rules

Whitespace is ignored except inside strings. Line comments start with `//` and continue to the end of the line.

Identifiers start with a Unicode letter or `_`, followed by Unicode letters, digits, or `_`.

Reserved words:

```text
and class const else export false fun for if nil or print return
super this true var while break continue
```

## Values And Types

Glox has these runtime value categories:

| Type | Examples |
| --- | --- |
| `nil` | `nil` |
| boolean | `true`, `false` |
| number | `123`, `3.14`, `1e3`, `1.5e-2` |
| string | `"hello"`, `"a\nb"` |
| list | `[1, "x", true]` |
| map | `{name: "lox", "answer": 42}` |
| function | `fun add(a, b) { return a + b; }` |
| class / instance | `class Box {}` / `Box()` |
| module | `import("math")` |
| native function | functions provided by Go |

Only `nil` and `false` are falsey. Everything else is truthy, including `0`, `""`, empty lists, and empty maps.

```lox
print !nil;    // true
print !false;  // true
print !0;      // false
print !"";     // false
```

### Numbers

Integer literals are stored as `int64`. Decimal and scientific notation literals are stored as `float64`.

```lox
var i = 123;      // int64
var f = 1.25;     // float64
var s = 1e3;      // float64
var t = 1.5e-2;   // float64
```

When both operands are integers, arithmetic uses integer operations. When either operand is a float, both operands are converted to float.

```lox
print 5 / 2;      // 2
print 5 / 2.0;    // 2.5
print 1 + 2;      // 3
print 1 + 2.5;    // 3.5
```

Number equality compares numeric value across `int64` and `float64`:

```lox
print 1 == 1.0;   // true
```

Integer division by zero is a runtime error. Floating-point division follows Go `float64` behavior, so `1.0 / 0.0` produces infinity.

### Strings

Strings use double quotes. Supported escapes:

| Escape | Meaning |
| --- | --- |
| `\"` | quote |
| `\\` | backslash |
| `\n` | newline |
| `\r` | carriage return |
| `\t` | tab |
| `\b` | backspace |

Unknown escapes keep the escaped character:

```lox
print "a\nb";
```

## Expressions

### Operators

| Category | Operators |
| --- | --- |
| grouping | `(expr)` |
| call | `fn(args)` |
| property | `object.name`, `object.name = value` |
| index | `list[i]`, `map[key]`, `string[i]`, `list[i] = value`, `map[key] = value` |
| unary | `!expr`, `-expr`, `++name`, `--name` |
| factor | `*`, `/` |
| term | `+`, `-` |
| comparison | `<`, `<=`, `>`, `>=` |
| equality | `==`, `!=` |
| logical | `and`, `or` |
| ternary | `condition ? thenExpr : elseExpr` |
| assignment | `name = value` |

`and` and `or` short-circuit and return the selected operand.

```lox
fun yes() {
  print "yes";
  return true;
}

print false and yes();  // yes() is not called
print true or yes();    // yes() is not called
```

The ternary operator is expression-level:

```lox
var label = score >= 60 ? "pass" : "fail";
```

### Increment And Decrement

Prefix and postfix increments work on variables:

```lox
var a = 1;
print a++;  // 1
print a;    // 2
print ++a;  // 3
print a--;  // 3
print --a;  // 1
```

They assign back to the variable, so they cannot target `const` bindings.

## Statements And Declarations

### Expression Statements

Expression statements may end with `;`. A final expression at end-of-file may omit `;`, which is useful with the embedding API.

```lox
2 + 2;
2 + 2
```

Both evaluate silently in normal script execution. Use `print` to write output:

```lox
print 2 + 2;  // 4
```

### Variables

`var` declares mutable variables. Missing initializers become `nil`. Multiple declarations can appear in one statement.

```lox
var a;
var b = 2;
var c = 1, d = 2, e;
```

### Constants

`const` declares immutable bindings. A const declaration must have an initializer.

```lox
const answer = 42;
answer = 43;  // runtime or compile error, depending on scope
```

Const-ness is preserved through closures:

```lox
fun outer() {
  const x = 1;
  fun inner() {
    x = 2; // error
  }
  return inner;
}
```

### Blocks And Scope

Blocks introduce lexical scopes:

```lox
var name = "global";
{
  var name = "local";
  print name; // local
}
print name;   // global
```

Top-level bindings live in the current module environment.

### If

```lox
if (condition) {
  print "then";
} else {
  print "else";
}
```

The parentheses around the condition are required.

### While

```lox
var i = 0;
while (i < 3) {
  print i;
  i = i + 1;
}
```

### For

The `for` loop has C-style clauses. Each clause is optional.

```lox
for (var i = 0; i < 3; i = i + 1) {
  print i;
}

for (;;) {
  print "once";
  break;
}
```

The initializer may be `var`, `const`, an expression statement, or empty.

### Break And Continue

`break` exits the nearest loop. `continue` jumps to the next iteration.

```lox
while (true) {
  if (done) break;
  if (skip) continue;
}
```

Both are only valid inside loops.

### Return

`return` is valid only inside functions and methods.

```lox
fun add(a, b) {
  return a + b;
}
```

Returning a value from an initializer is an error.

## Functions And Closures

Function declarations:

```lox
fun add(a, b) {
  return a + b;
}

print add(2, 3);
```

Anonymous functions use `fun` as an expression:

```lox
var add = fun(a, b) {
  return a + b;
};
```

Functions are closures and capture lexical variables:

```lox
fun makeCounter() {
  var count = 0;
  fun inc() {
    count = count + 1;
    return count;
  }
  return inc;
}

var c = makeCounter();
print c(); // 1
print c(); // 2
```

Functions check arity at runtime.

## Classes

Classes are callable. Calling a class creates an instance. If the class has an `init` method, it is called as the initializer.

```lox
class Box {
  init(value) {
    this.value = value;
  }

  get() {
    return this.value;
  }
}

var box = Box(42);
print box.get(); // 42
```

Fields are dynamic and live on instances:

```lox
box.label = "answer";
print box.label;
```

Methods can be extracted and keep their receiver:

```lox
var method = box.get;
print method();
```

### Inheritance

Use `<` to inherit from a superclass.

```lox
class Doughnut {
  describe() {
    return "doughnut";
  }
}

class Boston < Doughnut {
  describe() {
    return super.describe() + ":cream";
  }
}

print Boston().describe();
```

`this` is valid only inside classes. `super` is valid only in subclasses.

## Lists

Lists are ordered, mutable collections.

```lox
var xs = [1, 2, 3];
print xs[0];  // 1
xs[1] = 9;
print xs;     // [1, 9, 3]
```

List indices are zero-based integers.

Built-in list methods:

| Method | Description |
| --- | --- |
| `push(value)` | Append `value`; returns `true`. |
| `pop()` | Remove and return the last item; returns `nil` if empty. |
| `size()` | Return item count as an integer. |
| `insertAt(index, value)` | Insert before `index`; returns `true`. |
| `remove(index)` | Remove and return item at `index`. |

```lox
var xs = [];
xs.push("a");
xs.push("c");
xs.insertAt(1, "b");
print xs.remove(0); // a
print xs.size();    // 2
```

## Maps

Maps are mutable string-keyed collections.

```lox
var key = "dynamic";
var m = {
  name: "glox",
  "version": 1,
  [key]: true
};
```

Map keys must be strings. Identifier keys in literals are converted to strings.

Access and assignment:

```lox
print m.name;
print m["version"];
m.enabled = true;
m["count"] = 3;
```

Built-in map methods:

| Method | Description |
| --- | --- |
| `size()` | Return number of entries. |
| `keys()` | Return a list of keys. |
| `has(key)` | Return whether `key` exists. |
| `remove(key)` | Remove `key` and return the removed value. |

Map string output sorts keys for stable display.

## Strings As Indexed Values

Strings support indexing by zero-based Unicode rune index:

```lox
print "glox"[0];   // g
print "你好"[1];   // 好
```

String indexes are read-only.

## Modules And Imports

The global `import(path)` function loads modules. Imported modules are cached, so importing the same resolved module multiple times returns the same module object.

```lox
var a = import("hello.lox");
var b = import("./something/a.lox");
var c = import("/math/b.lox");
```

Path behavior:

| Import path | Meaning |
| --- | --- |
| `"name.lox"` | Search caller directory, configured module paths, then root directory. |
| `"name"` | Same as above, with `.lox` tried automatically for files; standard library names resolve to native modules. |
| `"./x.lox"` / `"../x.lox"` | Relative to the importing file's directory, or root directory in REPL. |
| `"/x.lox"` | Rooted at the VM `RootDir`, not necessarily the OS filesystem root. |
| OS absolute path | Loaded directly. |

Circular imports are detected and reported as errors.

### Exports

Only exported names are visible through a module object.

```lox
export var value = 3;
export const name = "module";

export fun add(a, b) {
  return a + b;
}

export class Box {
  init(v) {
    this.v = v;
  }
}
```

You can also export names after declaration:

```lox
var c = "late";
class C {}
fun ok() {
  return "ok";
}

export {
  c, C, ok
}
```

Non-exported names are private to the module:

```lox
var hidden = 1;
export var shown = 2;

// import("m.lox").hidden is an error.
```

Exported mutable variables can be assigned through the module object:

```lox
var m = import("counter.lox");
m.counter = m.counter + 1;
```

Assigning a non-exported property or assigning to an exported const is an error.

## Built-In Globals

These names are always available:

| Name | Description |
| --- | --- |
| `clock()` | Current Unix time in seconds as a float. |
| `len(value)` | Length of string, list, map, or module exports. Unknown values return `0`. |
| `type(value)` | Type name string. |
| `string(value)` | Convert to Glox display string. |
| `import(path)` | Import a Lox file, native module, or standard library module. |

Type names returned by `type` include:

```text
nil boolean number string function native-function class object list map module unknown
```

## Standard Library

Standard library modules are native modules registered by the VM. They do not rely on `.lox` wrapper files, so compiled executables can run from any working directory.

Import forms supported for standard modules:

```lox
var math = import("math");
var math2 = import("math.lox");
var math3 = import("stdlib/math");
var math4 = import("stdlib/math.lox");
```

All four refer to the same cached native module.

Current standard modules:

```text
std os string json math times regexp
```

### `std`

Import:

```lox
var std = import("std");
```

Functions:

| Name | Description |
| --- | --- |
| `assert(value)` | Error if `value` is falsey; returns `true` otherwise. |
| `assert(value, message)` | Error with `message` if `value` is falsey. |
| `bool(value)` | Return truthiness as a boolean. |
| `clone(value)` | Shallow-copy lists and maps; other values are returned unchanged. |
| `eprint(...)` | Write values to stderr without newline. |
| `eprintln(...)` | Write values to stderr with newline. |
| `has(mapOrModule, key)` | Return whether a map or module export contains `key`. |
| `isBoolean(value)` | Type predicate. |
| `isList(value)` | Type predicate. |
| `isMap(value)` | Type predicate. |
| `isModule(value)` | Type predicate. |
| `isNil(value)` | Type predicate. |
| `isNumber(value)` | Type predicate. |
| `isString(value)` | Type predicate. |
| `keys(mapOrModule)` | Return sorted key list. |
| `number(value)` | Return a number or parse a numeric string. |
| `panic(message)` | Raise a runtime error. |
| `print(...)` | Write values to stdout without newline, separated by spaces. |
| `println(...)` | Write values to stdout with newline, separated by spaces. |
| `range(end)` | Return `[0, 1, ..., end - 1]`. |
| `range(start, end)` | Return values from `start` to `end`, excluding `end`. |
| `range(start, end, step)` | Return stepped integer range. `step` cannot be zero. |
| `string(value)` | Convert to display string. |
| `typeOf(value)` | Return type name string. |
| `values(mapOrModule)` | Return values ordered by sorted keys. |

Example:

```lox
var std = import("std");

std.assert(1 < 2, "math still works");
print std.range(1, 5, 2); // [1, 3]
print std.keys({b: 2, a: 1}); // [a, b]
```

### `os`

Import:

```lox
var os = import("os");
```

Exports:

| Name | Description |
| --- | --- |
| `PathSeparator` | Host OS path separator as a string. |
| `abs(path)` | Absolute path. |
| `args()` | Script arguments from `glox script.lox args...`. |
| `base(path)` | Last path element. |
| `clean(path)` | Clean path. |
| `cwd()` | Current working directory. |
| `dir(path)` | Directory portion of path. |
| `exists(path)` | Whether path exists. |
| `ext(path)` | File extension. |
| `getenv(key)` | Environment value or `""`. |
| `isAbs(path)` | Whether path is absolute. |
| `isDir(path)` | Whether path exists and is a directory. |
| `isFile(path)` | Whether path exists and is a regular file. |
| `join(...)` | Join path parts. |
| `mkdirAll(path)` | Create directory tree with default permissions; returns `true`. |
| `readFile(path)` | Read file as string. |
| `remove(path)` | Remove file or empty directory; returns `true`. |
| `setenv(key, value)` | Set environment variable; returns `true`. |
| `tempDir()` | Host temp directory. |
| `unsetenv(key)` | Unset environment variable; returns `true`. |
| `writeFile(path, content)` | Write string to file; returns `true`. |

Example:

```lox
var os = import("os");

var path = os.join(os.tempDir(), "glox-note.txt");
os.writeFile(path, "hello");
print os.readFile(path);
os.remove(path);
```

### `string`

Import:

```lox
var string = import("string");
```

Functions use Unicode rune counts for `len`, `index`, `lastIndex`, and `slice`.

| Name | Description |
| --- | --- |
| `contains(text, substr)` | Whether `text` contains `substr`. |
| `fields(text)` | Split around runs of whitespace. |
| `hasPrefix(text, prefix)` | Prefix test. |
| `hasSuffix(text, suffix)` | Suffix test. |
| `index(text, substr)` | First rune index, or `-1`. |
| `join(list, separator)` | Join values after converting each item with Glox stringification. |
| `lastIndex(text, substr)` | Last rune index, or `-1`. |
| `len(text)` | Rune count. |
| `lower(text)` | Lowercase. |
| `repeat(text, count)` | Repeat `count` times. |
| `replace(text, old, new, count)` | Replace up to `count` occurrences. Use `-1` for all. |
| `split(text, separator)` | Split into a list of strings. |
| `slice(text, start, end)` | Rune slice `[start, end)`. |
| `trim(text, cutset)` | Trim leading/trailing cutset runes. |
| `trimPrefix(text, prefix)` | Trim prefix. |
| `trimSpace(text)` | Trim Unicode whitespace. |
| `trimSuffix(text, suffix)` | Trim suffix. |
| `upper(text)` | Uppercase. |

Example:

```lox
var string = import("string");

print string.len("你a");                  // 2
print string.slice("abcdef", 1, 4);       // bcd
print string.join(["x", 2, true], "-");   // x-2-true
```

### `json`

Import:

```lox
var json = import("json");
```

Functions:

| Name | Description |
| --- | --- |
| `compact(jsonText)` | Remove insignificant whitespace. |
| `decode(jsonText)` | Decode JSON into Glox values. Objects become maps; arrays become lists. |
| `encode(value)` | Encode nil, booleans, numbers, strings, lists, maps, or module exports. |
| `indent(jsonText, prefix, indent)` | Pretty-print JSON. |
| `valid(jsonText)` | Return whether text is valid JSON. |

Integer JSON numbers decode as `int64` when possible; decimal or exponent numbers decode as `float64`.

Example:

```lox
var json = import("json");

var data = json.decode("{\"name\":\"glox\",\"scores\":[1,2,3]}");
print data.name;
print data.scores[1];
print json.encode({ok: true, value: 3});
```

`encode` cannot encode functions, classes, instances, native functions, or non-finite floats.

### `math`

Import:

```lox
var math = import("math");
```

Constants:

| Name | Description |
| --- | --- |
| `PI` | Pi. |
| `E` | Euler's number. |
| `MaxInt` | Maximum `int64`. |
| `MinInt` | Minimum `int64`. |

Functions:

| Name | Description |
| --- | --- |
| `abs(value)` | Absolute value. |
| `ceil(value)` | Ceiling. |
| `clamp(value, low, high)` | Clamp into `[low, high]`. |
| `cos(value)` | Cosine. |
| `exp(value)` | Exponential. |
| `floor(value)` | Floor. |
| `isInf(value)` | Whether value is infinity. |
| `isNaN(value)` | Whether value is NaN. |
| `log(value)` | Natural logarithm. |
| `log10(value)` | Base-10 logarithm. |
| `max(...)` | Maximum of one or more numbers. |
| `min(...)` | Minimum of one or more numbers. |
| `mod(left, right)` | Remainder. Integer inputs use integer remainder. |
| `pow(base, exponent)` | Power. |
| `round(value)` | Round to nearest integer. |
| `sin(value)` | Sine. |
| `sqrt(value)` | Square root. |
| `tan(value)` | Tangent. |
| `trunc(value)` | Truncate toward zero. |

Many math functions return an integer when the floating-point result is exactly representable as `int64`.

Example:

```lox
var math = import("math");

print math.sqrt(9);      // 3
print math.pow(2, 3);    // 8
print math.mod(10, 4);   // 2
```

### `times`

Import:

```lox
var times = import("times");
```

Time values are represented as Unix seconds. Whole-second times are returned as integers; sub-second times are returned as floats.

Layout constants:

| Name | Value |
| --- | --- |
| `ANSIC` | Go `time.ANSIC` layout. |
| `DateOnly` | `2006-01-02` |
| `DateTime` | `2006-01-02 15:04:05` |
| `Kitchen` | Go `time.Kitchen` layout. |
| `RFC3339` | Go `time.RFC3339` layout. |
| `RFC3339Nano` | Go `time.RFC3339Nano` layout. |
| `TimeOnly` | `15:04:05` |

Functions:

| Name | Description |
| --- | --- |
| `add(time, seconds)` | Add seconds and return a time value. |
| `date(year, month, day, hour, minute, second)` | Create UTC time value. |
| `format(time, layout)` | Format UTC time with Go layout. |
| `formatNow(layout)` | Format current UTC time. |
| `now()` | Current Unix seconds as float. |
| `parse(layout, value)` | Parse text with Go layout and return time value. |
| `since(time)` | Seconds since time. |
| `sleep(seconds)` | Sleep for non-negative seconds; returns `true`. |
| `unix()` | Current Unix seconds as integer. |
| `unixMilli()` | Current Unix milliseconds as integer. |
| `unixNano()` | Current Unix nanoseconds as integer. |

Example:

```lox
var times = import("times");

var epoch = times.parse(times.RFC3339, "1970-01-01T00:00:00Z");
print epoch;                              // 0
print times.format(epoch, times.RFC3339); // 1970-01-01T00:00:00Z
print times.add(epoch, 60);               // 60
```

### `regexp`

Import:

```lox
var regexp = import("regexp");
```

Patterns use Go regular expression syntax.

| Name | Description |
| --- | --- |
| `find(pattern, text)` | First match string, or `nil`. |
| `findAll(pattern, text, limit)` | List of matches. `limit` follows Go regexp semantics; `-1` means all. |
| `findSubmatch(pattern, text)` | List containing full match and capture groups, or `nil`. |
| `match(pattern, text)` | Boolean match test. |
| `quoteMeta(text)` | Escape text for literal matching. |
| `replace(pattern, text, replacement)` | Replace all matches. |
| `split(pattern, text, limit)` | Split text by pattern. |

Example:

```lox
var regexp = import("regexp");

print regexp.match("^[a-z]+$", "abc");           // true
print regexp.find("[0-9]+", "a12b");             // 12
print regexp.findSubmatch("([a-z]+)([0-9]+)", "ab12")[2]; // 12
print regexp.replace("[0-9]+", "a12b", "#");     // a#b
```

## Embedding Glox In Go

Create a VM with `NewVM`:

```go
vm := glox.NewVM(glox.Options{})
```

Useful options:

| Option | Description |
| --- | --- |
| `Stdout` | Writer for `print` and stdout standard library functions. |
| `Stderr` | Writer for diagnostics and stderr standard library functions. |
| `RootDir` | Base directory for rooted imports like `"/math/b.lox"`. |
| `ModulePaths` | Extra module search directories. |
| `Args` | Values returned by `import("os").args()`. |
| `DebugWriter` | Writer for disassembly / trace output. |
| `DebugDisassemble` | Disassemble compiled functions. |
| `DebugTraceExecution` | Trace bytecode execution. |

### Execute Source

```go
vm := glox.NewVM(glox.Options{})

err := vm.RunString(`print "hello";`)
```

Use `DoString` to execute and retrieve the last expression-statement result:

```go
vm := glox.NewVM(glox.Options{})

value, err := vm.DoString(`
var a = 3;
a = a + 1
`)
// value is int64(4)
```

The final expression may omit `;`, but expression statements with `;` also update the VM's last result.

### Execute Files

```go
err := vm.RunFile("main.lox")

value, err := vm.DoFile("main.lox")
```

### Globals

```go
vm := glox.NewVM(glox.Options{})

vm.SetGlobal("host", int64(10))

value, err := vm.DoString(`var a = 3; a = a + host`)
// value is int64(13)

a, ok := vm.GetGlobal("a")
```

`GetGlobal` checks the last executed module, then the REPL module, then VM global builtins.

### Defining Native Functions

```go
vm.DefineNative("hostAdd", 2, func(vm *glox.VM, args []glox.Value) (glox.Value, error) {
  left, ok := glox.AsFloat64(args[0])
  if !ok {
    return nil, fmt.Errorf("left must be a number")
  }
  right, ok := glox.AsFloat64(args[1])
  if !ok {
    return nil, fmt.Errorf("right must be a number")
  }
  return left + right, nil
})
```

From Lox:

```lox
print hostAdd(2, 3);
```

### Defining Native Modules

```go
vm.RegisterNativeModule("host", map[string]glox.Value{
  "name": "native",
  "add": &glox.NativeFunction{
    Name:  "host.add",
    Arity: 2,
    Fn: func(vm *glox.VM, args []glox.Value) (glox.Value, error) {
      left, _ := glox.AsFloat64(args[0])
      right, _ := glox.AsFloat64(args[1])
      return left + right, nil
    },
  },
})
```

From Lox:

```lox
var host = import("host");
print host.name;
print host.add(3, 4);
```

### Go Value Mapping

Native functions can return these values directly:

| Go value | Glox value |
| --- | --- |
| `nil` | `nil` |
| `bool` | boolean |
| `int64` | integer number |
| `float64` | floating number |
| `string` | string |
| `*glox.List` | list |
| `*glox.Map` | map |
| `*glox.Module` | module |
| `*glox.NativeFunction` | callable native function |

Use helpers:

```go
glox.IsNumber(value)
glox.AsInt64(value)
glox.AsFloat64(value)
glox.Stringify(value)
glox.ValuesEqual(a, b)
```

Native errors become Glox runtime errors with stack traces.

## Bytecode Debugging

Glox has a Go-style disassembler and execution trace integration.

```go
var debug bytes.Buffer
vm := glox.NewVM(glox.Options{
  DebugWriter:         &debug,
  DebugDisassemble:    true,
  DebugTraceExecution: true,
})
```

`DebugDisassemble` writes bytecode for compiled functions. `DebugTraceExecution` writes stack and instruction traces during execution.

## Error Model

Glox reports compile-time diagnostics through `Diagnostics` and runtime failures as runtime errors. In the command-line runner, reported diagnostic/runtime errors are not printed twice.

Common runtime errors include:

| Error kind | Example |
| --- | --- |
| Undefined variable | `print missing;` |
| Const assignment | `const x = 1; x = 2;` |
| Invalid property | `1.name` |
| Invalid index | `[1][3]` |
| Hidden export | `import("m.lox").hidden` |
| Circular import | mutually importing modules |
| Native argument error | `math.sqrt("x")` |

Glox does not currently have `try/catch` or user-defined exceptions.

## Complete Example

```lox
var std = import("std");
var string = import("string");
var json = import("json");
var math = import("math");

class Report {
  init(title) {
    this.title = title;
    this.rows = [];
  }

  add(name, score) {
    this.rows.push({name: name, score: score});
  }

  average() {
    var total = 0;
    for (var i = 0; i < this.rows.size(); i = i + 1) {
      total = total + this.rows[i].score;
    }
    return total / this.rows.size();
  }
}

var report = Report("scores");
report.add("ana", 10);
report.add("ben", 8);

print string.upper(report.title);
print math.round(report.average());
print json.encode({
  title: report.title,
  average: report.average(),
  rows: report.rows
});

std.assert(report.average() >= 9, "average too low");
```
