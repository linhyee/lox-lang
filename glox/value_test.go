package glox

import (
	"math"
	"testing"
)

func TestValueTruthinessEqualityAndStringify(t *testing.T) {
	if !IsFalsey(nil) || !IsFalsey(false) || IsFalsey(true) || IsFalsey(int64(0)) || IsFalsey(float64(0)) || IsFalsey("") {
		t.Fatal("falsey semantics mismatch")
	}

	cases := []struct {
		a, b Value
		want bool
	}{
		{nil, nil, true},
		{nil, false, false},
		{true, true, true},
		{true, false, false},
		{int64(1), int64(1), true},
		{int64(1), int64(2), false},
		{int64(1), float64(1), true},
		{float64(1), int64(1), true},
		{float64(1), float64(1), true},
		{float64(1), float64(2), false},
		{"x", "x", true},
		{"x", "y", false},
	}
	for _, tc := range cases {
		if got := ValuesEqual(tc.a, tc.b); got != tc.want {
			t.Fatalf("ValuesEqual(%#v,%#v)=%v want %v", tc.a, tc.b, got, tc.want)
		}
	}

	list := &List{Items: []Value{int64(1), "x", nil, true}}
	m := NewMap()
	m.Items["b"] = float64(2)
	m.Items["a"] = "one"
	stringCases := map[Value]string{
		nil:                  "nil",
		true:                 "true",
		false:                "false",
		int64(12):            "12",
		float64(12):          "12",
		float64(12.25):       "12.25",
		math.Inf(1):          "+Inf",
		"hello":              "hello",
		list:                 "[1, x, nil, true]",
		m:                    "{a: one, b: 2}",
		&Function{Name: "f"}: "<fn f>",
	}
	for value, want := range stringCases {
		if got := Stringify(value); got != want {
			t.Fatalf("Stringify(%#v)=%q want %q", value, got, want)
		}
	}
}
