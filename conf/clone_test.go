package conf

import (
	"reflect"
	"testing"
)

type cloneNode struct {
	Labels map[string]string
	Rules  []int
	Next   *cloneNode
}

type hiddenRef struct {
	m map[string]int
}

type visibleRef struct {
	M map[string]int
}

type optionalInt *int

func TestHasReference(t *testing.T) {
	_ = hiddenRef{m: nil}.m
	tests := []struct {
		name string
		typ  reflect.Type
		want bool
	}{
		{name: "int", typ: reflect.TypeFor[int](), want: false},
		{name: "string", typ: reflect.TypeFor[string](), want: false},
		{name: "map", typ: reflect.TypeFor[map[string]int](), want: true},
		{name: "slice", typ: reflect.TypeFor[[]int](), want: true},
		{name: "pointer", typ: reflect.TypeFor[*int](), want: true},
		{name: "interface", typ: reflect.TypeFor[any](), want: true},
		{name: "struct with map", typ: reflect.TypeFor[cloneNode](), want: true},
		{name: "struct with unexported map", typ: reflect.TypeFor[hiddenRef](), want: false},
		{name: "struct with exported map", typ: reflect.TypeFor[visibleRef](), want: true},
		{name: "array of int", typ: reflect.TypeFor[[2]int](), want: false},
		{name: "array of map", typ: reflect.TypeFor[[2]map[string]int](), want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasReference(tt.typ); got != tt.want {
				t.Errorf("hasReference(%s) = %t, want %t", tt.typ, got, tt.want)
			}
		})
	}
}

func TestCloneGetterValue(t *testing.T) {
	t.Run("map", func(t *testing.T) {
		in := map[string]int{"a": 1}
		got := cloneGetterValue(in)
		got["a"] = 2
		if in["a"] != 1 {
			t.Errorf("source map changed to %d, want 1", in["a"])
		}
	})

	t.Run("slice", func(t *testing.T) {
		in := []int{1}
		got := cloneGetterValue(in)
		got[0] = 2
		if in[0] != 1 {
			t.Errorf("source slice changed to %d, want 1", in[0])
		}
	})

	t.Run("pointer", func(t *testing.T) {
		in := &cloneNode{Labels: map[string]string{"a": "1"}}
		got := cloneGetterValue(in)
		got.Labels["a"] = "2"
		if in.Labels["a"] != "1" {
			t.Errorf("source pointer changed to %q, want 1", in.Labels["a"])
		}
	})

	t.Run("defined pointer", func(t *testing.T) {
		number := 7
		in := optionalInt(&number)
		got := cloneGetterValue(in)
		if got == nil {
			t.Fatal("defined pointer became nil")
		}
		if got == in {
			t.Error("defined pointer was not copied")
		}
		if *got != 7 {
			t.Errorf("defined pointer = %d, want 7", *got)
		}
		*got = 8
		if number != 7 {
			t.Errorf("source number changed to %d, want 7", number)
		}
	})

	t.Run("nested struct", func(t *testing.T) {
		in := cloneNode{Labels: map[string]string{"a": "1"}, Rules: []int{1}}
		got := cloneGetterValue(in)
		got.Labels["a"] = "2"
		got.Rules[0] = 2
		if in.Labels["a"] != "1" {
			t.Errorf("source labels changed to %q, want 1", in.Labels["a"])
		}
		if in.Rules[0] != 1 {
			t.Errorf("source rule changed to %d, want 1", in.Rules[0])
		}
	})

	t.Run("shared pointer", func(t *testing.T) {
		shared := &cloneNode{}
		in := struct {
			A *cloneNode
			B *cloneNode
		}{A: shared, B: shared}
		got := cloneGetterValue(in)
		if got.A != got.B {
			t.Error("shared pointer was split into two copies")
		}
		if got.A == shared {
			t.Error("pointer was not copied")
		}
	})

	t.Run("cycle", func(t *testing.T) {
		in := &cloneNode{}
		in.Next = in
		got := cloneGetterValue(in)
		if got == in {
			t.Fatal("pointer was not copied")
		}
		if got.Next != got {
			t.Error("cycle was not preserved in the copy")
		}
	})

	t.Run("empty containers stay non-nil", func(t *testing.T) {
		base := []int{1}
		if got := cloneGetterValue(base[:0]); got == nil {
			t.Error("empty slice became nil")
		}
		if got := cloneGetterValue(map[string]int{}); got == nil {
			t.Error("empty map became nil")
		}
	})

	t.Run("nil containers stay nil", func(t *testing.T) {
		var slice []int
		if got := cloneGetterValue(slice); got != nil {
			t.Errorf("nil slice = %v, want nil", got)
		}
		var mp map[string]int
		if got := cloneGetterValue(mp); got != nil {
			t.Errorf("nil map = %v, want nil", got)
		}
	})
}
