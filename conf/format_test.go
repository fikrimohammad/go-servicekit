package conf

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

type textValue struct {
	V string
}

func (t *textValue) UnmarshalText(b []byte) error {
	t.V = string(b)
	return nil
}

type jsonDoc struct {
	V string
}

func (j *jsonDoc) UnmarshalJSON(b []byte) error {
	j.V = "json:" + string(b)
	return nil
}

type yamlDoc struct {
	V string
}

func (y *yamlDoc) UnmarshalYAML(n *yaml.Node) error {
	y.V = "yaml:" + n.Value
	return nil
}

func TestFormatValidate(t *testing.T) {
	tests := []struct {
		name    string
		format  Format
		wantErr error
	}{
		{name: "yaml", format: FormatYAML},
		{name: "json", format: FormatJSON},
		{name: "toml", format: FormatTOML},
		{name: "scalar", format: FormatScalar},
		{name: "empty", format: "", wantErr: ErrInvalidFormat},
		{name: "unknown", format: "xml", wantErr: ErrInvalidFormat},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.format.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Format(%q).Validate() = %v, want %v", tt.format, err, tt.wantErr)
			}
		})
	}
}

func TestParseValue(t *testing.T) {
	type pair struct {
		Host string `yaml:"host" json:"host" toml:"host"`
		Port int    `yaml:"port" json:"port" toml:"port"`
	}

	tests := []struct {
		name    string
		raw     string
		typ     reflect.Type
		format  Format
		want    any
		wantErr error
	}{
		{
			name:   "yaml/struct",
			raw:    "host: localhost\nport: 5432\n",
			typ:    reflect.TypeFor[pair](),
			format: FormatYAML,
			want:   pair{Host: "localhost", Port: 5432},
		},
		{
			name:   "yaml/duration",
			raw:    "30s",
			typ:    reflect.TypeFor[time.Duration](),
			format: FormatYAML,
			want:   30 * time.Second,
		},
		{
			name:   "json/struct",
			raw:    `{"host":"localhost","port":5432}`,
			typ:    reflect.TypeFor[pair](),
			format: FormatJSON,
			want:   pair{Host: "localhost", Port: 5432},
		},
		{
			name:   "json/large int",
			raw:    "9007199254740993",
			typ:    reflect.TypeFor[int64](),
			format: FormatJSON,
			want:   int64(9007199254740993),
		},
		{
			name:   "toml/struct",
			raw:    "host = \"localhost\"\nport = 5432\n",
			typ:    reflect.TypeFor[pair](),
			format: FormatTOML,
			want:   pair{Host: "localhost", Port: 5432},
		},
		{
			name:   "toml/scalar",
			raw:    "30",
			typ:    reflect.TypeFor[int](),
			format: FormatTOML,
			want:   30,
		},
		{
			name:   "toml/array",
			raw:    "[1, 2]",
			typ:    reflect.TypeFor[[]int](),
			format: FormatTOML,
			want:   []int{1, 2},
		},
		{
			name:   "scalar/string",
			raw:    "yes",
			typ:    reflect.TypeFor[string](),
			format: FormatScalar,
			want:   "yes",
		},
		{
			name:   "scalar/bool",
			raw:    "true",
			typ:    reflect.TypeFor[bool](),
			format: FormatScalar,
			want:   true,
		},
		{
			name:   "scalar/int",
			raw:    "-42",
			typ:    reflect.TypeFor[int](),
			format: FormatScalar,
			want:   int(-42),
		},
		{
			name:   "scalar/uint",
			raw:    "42",
			typ:    reflect.TypeFor[uint](),
			format: FormatScalar,
			want:   uint(42),
		},
		{
			name:   "scalar/float",
			raw:    "1.5",
			typ:    reflect.TypeFor[float64](),
			format: FormatScalar,
			want:   1.5,
		},
		{
			name:   "scalar/duration",
			raw:    "30s",
			typ:    reflect.TypeFor[time.Duration](),
			format: FormatScalar,
			want:   30 * time.Second,
		},
		{
			name:   "scalar/text unmarshaler",
			raw:    "hello",
			typ:    reflect.TypeFor[textValue](),
			format: FormatScalar,
			want:   textValue{V: "hello"},
		},
		{
			name:    "yaml/bad",
			raw:     "not: [valid",
			typ:     reflect.TypeFor[pair](),
			format:  FormatYAML,
			wantErr: ErrParse,
		},
		{
			name:    "json/bad",
			raw:     "{",
			typ:     reflect.TypeFor[pair](),
			format:  FormatJSON,
			wantErr: ErrParse,
		},
		{
			name:    "toml/bad",
			raw:     "host =",
			typ:     reflect.TypeFor[pair](),
			format:  FormatTOML,
			wantErr: ErrParse,
		},
		{
			name:    "scalar/bad int",
			raw:     "not-an-int",
			typ:     reflect.TypeFor[int](),
			format:  FormatScalar,
			wantErr: ErrParse,
		},
		{
			name:    "scalar/unsupported",
			raw:     "value",
			typ:     reflect.TypeFor[pair](),
			format:  FormatScalar,
			wantErr: ErrInvalidType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseValue(tt.raw, tt.typ, tt.format)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("parseValue(%q, %s, %s) error = %v, want %v", tt.raw, tt.typ, tt.format, err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if !equalValue(got, tt.want) {
				t.Errorf("parseValue(%q, %s, %s) = %#v, want %#v", tt.raw, tt.typ, tt.format, got, tt.want)
			}
		})
	}
}

func equalValue(a, b any) bool {
	if av, ok := a.([]int); ok {
		bv, ok := b.([]int)
		return ok && slices.Equal(av, bv)
	}
	return a == b
}

func TestParseValueCustomUnmarshalers(t *testing.T) {
	got, err := parseValue(`{"a":1}`, reflect.TypeFor[jsonDoc](), FormatJSON)
	if err != nil {
		t.Fatalf("parseValue JSON: %v", err)
	}
	if want := "json:{\"a\":1}"; got.(jsonDoc).V != want {
		t.Errorf("JSON unmarshaler = %q, want %q", got.(jsonDoc).V, want)
	}

	got, err = parseValue("hello", reflect.TypeFor[yamlDoc](), FormatYAML)
	if err != nil {
		t.Fatalf("parseValue YAML: %v", err)
	}
	if want := "yaml:hello"; got.(yamlDoc).V != want {
		t.Errorf("YAML unmarshaler = %q, want %q", got.(yamlDoc).V, want)
	}
}

func TestParseValueTOMLDate(t *testing.T) {
	type doc struct {
		D toml.LocalDate `toml:"d"`
	}

	got, err := parseValue("d = 2026-11-08\n", reflect.TypeFor[doc](), FormatTOML)
	if err != nil {
		t.Fatalf("parseValue TOML date: %v", err)
	}
	if want := "2026-11-08"; got.(doc).D.String() != want {
		t.Errorf("TOML local date = %q, want %q", got.(doc).D.String(), want)
	}
}

func TestParseValueErrorHidesRawValue(t *testing.T) {
	const secret = "supersecret-value"

	_, err := parseValue(secret, reflect.TypeFor[int](), FormatYAML)
	if !errors.Is(err, ErrParse) {
		t.Fatalf("parseValue() error = %v, want ErrParse", err)
	}
	if strings.Contains(err.Error(), secret) {
		t.Errorf("parseValue() error %q contains the raw value", err)
	}
}
