package conf

import (
	"errors"
	"testing"
)

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
