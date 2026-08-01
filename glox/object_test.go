package glox

import (
	"strings"
	"testing"
)

func TestObjectsStringAndLookup(t *testing.T) {
	function := &Function{Name: "foo", UpvalueCount: 2}
	if function.String() != "<fn foo>" {
		t.Fatalf("function string mismatch: %s", function)
	}
	if (&Function{}).String() != "<script>" {
		t.Fatalf("script function string mismatch")
	}

	module := NewModule("mod.lox")
	closure := NewClosure(function, module)
	if len(closure.Upvalues) != 2 || closure.Module != module || closure.String() != "<fn foo>" {
		t.Fatalf("closure mismatch: %#v", closure)
	}

	native := &NativeFunction{Name: "native", Arity: 0}
	if native.String() != "<native fn native>" {
		t.Fatalf("native string mismatch: %s", native)
	}
	boundNative := &BoundNativeFunction{Receiver: "r", Native: native}
	if boundNative.String() != native.String() {
		t.Fatalf("bound native string mismatch: %s", boundNative)
	}

	parentMethod := &Closure{Function: &Function{Name: "parent"}}
	childMethod := &Closure{Function: &Function{Name: "child"}}
	parent := &Class{Name: "Parent", Methods: map[string]Value{"shared": parentMethod}}
	child := &Class{Name: "Child", Superclass: parent, Methods: map[string]Value{"own": childMethod}}
	if child.String() != "Child" {
		t.Fatalf("class string mismatch: %s", child)
	}
	if child.FindMethod("own") != childMethod {
		t.Fatal("child method lookup failed")
	}
	if child.FindMethod("shared") != parentMethod {
		t.Fatal("superclass method lookup failed")
	}
	if child.FindMethod("missing") != nil {
		t.Fatal("missing method should be nil")
	}

	instance := NewInstance(child)
	if instance.String() != "Child instance" {
		t.Fatalf("instance string mismatch: %s", instance)
	}
	bound := &BoundMethod{Receiver: instance, Method: childMethod}
	if bound.String() != "<fn child>" {
		t.Fatalf("bound method string mismatch: %s", bound)
	}

	list := &List{Items: []Value{float64(1), "two"}}
	if list.String() != "[1, two]" {
		t.Fatalf("list string mismatch: %s", list)
	}
	m := NewMap()
	m.Items["b"] = float64(2)
	m.Items["a"] = float64(1)
	if m.String() != "{a: 1, b: 2}" {
		t.Fatalf("map string should be deterministic: %s", m)
	}
}

func TestEnvironmentAndModuleBindings(t *testing.T) {
	env := NewEnvironment()
	env.Define("x", float64(1), false)
	env.Define("k", "const", true)

	if value, ok := env.GetOwn("x"); !ok || value != float64(1) {
		t.Fatalf("GetOwn x mismatch: %#v %v", value, ok)
	}
	if binding, ok := env.Binding("k"); !ok || !binding.Const || binding.Value != "const" {
		t.Fatalf("Binding k mismatch: %#v %v", binding, ok)
	}
	if err := env.Assign("x", float64(2)); err != nil {
		t.Fatalf("assign x failed: %v", err)
	}
	if value, _ := env.GetOwn("x"); value != float64(2) {
		t.Fatalf("assign x did not update: %#v", value)
	}
	if err := env.Assign("k", "new"); err == nil || !strings.Contains(err.Error(), "cannot assign to const") {
		t.Fatalf("expected const assignment error, got %v", err)
	}
	if err := env.Assign("missing", nil); err == nil || !strings.Contains(err.Error(), "undefined variable") {
		t.Fatalf("expected missing assignment error, got %v", err)
	}

	module := NewModule("m.lox")
	module.Env.Define("shown", 3, false)
	module.Env.Define("hidden", 4, false)
	module.Export("shown")
	if !module.IsExported("shown") || module.IsExported("hidden") {
		t.Fatal("module export flags mismatch")
	}
	exported := module.Env.ExportedValues(module.Exports)
	if len(exported) != 1 || exported["shown"] != 3 {
		t.Fatalf("exported values mismatch: %#v", exported)
	}
}
