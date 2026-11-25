package conf

import (
	"errors"
	"reflect"
	"testing"
)

func TestGetterNoExportedFields(t *testing.T) {
	typ := reflect.TypeFor[Getter[int]]()
	for f := range typ.Fields() {
		if f.IsExported() {
			t.Errorf("Getter exposes field %s; callers could rebind it", f.Name)
		}
	}
}

func TestGetterCopy(t *testing.T) {
	g := boundGetter(7)
	copied := g
	if got, err := copied.Get(); err != nil || got != 7 {
		t.Errorf("copied Get() = %d, %v, want 7, nil", got, err)
	}
}

func TestGetterGetReturnsCopy(t *testing.T) {
	original := map[string]int{"a": 1}
	g := Getter[map[string]int]{}
	g.bind(func() (any, error) { return original, nil })

	got, err := g.Get()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	got["a"] = 2
	if original["a"] != 1 {
		t.Errorf("source map changed to %d, want 1", original["a"])
	}
}

func TestGetterGetWithDefaultReturnsCopy(t *testing.T) {
	original := []int{1}
	g := Getter[[]int]{}
	g.bind(func() (any, error) { return original, nil })

	got := g.GetWithDefault(nil)
	got[0] = 2
	if original[0] != 1 {
		t.Errorf("source slice changed to %d, want 1", original[0])
	}
}

func TestGetterGetNilValue(t *testing.T) {
	g := Getter[any]{}
	g.bind(func() (any, error) { return nil, nil })

	got, err := g.Get()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != nil {
		t.Errorf("Get() = %v, want nil", got)
	}
}

func TestGetterGetDefinedPointer(t *testing.T) {
	number := 7
	in := optionalInt(&number)
	g := Getter[optionalInt]{}
	g.bind(func() (any, error) { return in, nil })

	got, err := g.Get()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got == in {
		t.Error("defined pointer was not copied")
	}
	*got = 8
	if number != 7 {
		t.Errorf("source number changed to %d, want 7", number)
	}
}

var benchmarkValue any

func BenchmarkGetterGetInt(b *testing.B) {
	g := boundGetter(7)
	b.ReportAllocs()
	for b.Loop() {
		benchmarkValue, _ = g.Get()
	}
}

func BenchmarkGetterGetAnyInt(b *testing.B) {
	g := Getter[any]{}
	g.bind(func() (any, error) { return 7, nil })
	b.ReportAllocs()
	for b.Loop() {
		benchmarkValue, _ = g.Get()
	}
}

func BenchmarkGetterGetMap(b *testing.B) {
	g := Getter[map[string]int]{}
	g.bind(func() (any, error) { return map[string]int{"a": 1}, nil })
	b.ReportAllocs()
	for b.Loop() {
		benchmarkValue, _ = g.Get()
	}
}

func BenchmarkGetterGetAnyMap(b *testing.B) {
	g := Getter[any]{}
	g.bind(func() (any, error) { return map[string]int{"a": 1}, nil })
	b.ReportAllocs()
	for b.Loop() {
		benchmarkValue, _ = g.Get()
	}
}

func TestGetterGet(t *testing.T) {
	tests := []struct {
		name    string
		getter  Getter[int]
		want    int
		wantErr error
	}{
		{
			name:   "bound",
			getter: boundGetter(7),
			want:   7,
		},
		{
			name:    "unbound",
			getter:  Getter[int]{},
			wantErr: ErrUnbound,
		},
		{
			name:    "underlying error",
			getter:  boundGetterErr(ErrNotSet),
			wantErr: ErrNotSet,
		},
		{
			name:    "wrong type",
			getter:  boundGetterAny("not an int"),
			wantErr: ErrInvalidType,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.getter.Get()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Get() error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("Get() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestGetterGetWithDefault(t *testing.T) {
	tests := []struct {
		name   string
		getter Getter[int]
		want   int
	}{
		{name: "bound", getter: boundGetter(7), want: 7},
		{name: "unbound", getter: Getter[int]{}, want: 11},
		{name: "underlying error", getter: boundGetterErr(ErrNotSet), want: 11},
		{name: "wrong type", getter: boundGetterAny("not an int"), want: 11},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.getter.GetWithDefault(11); got != tt.want {
				t.Errorf("GetWithDefault(11) = %d, want %d", got, tt.want)
			}
		})
	}
}

func boundGetter(v int) Getter[int] {
	g := Getter[int]{}
	g.bind(func() (any, error) { return v, nil })
	return g
}

func boundGetterErr(err error) Getter[int] {
	g := Getter[int]{}
	g.bind(func() (any, error) { return nil, err })
	return g
}

func boundGetterAny(v any) Getter[int] {
	g := Getter[int]{}
	g.bind(func() (any, error) { return v, nil })
	return g
}
