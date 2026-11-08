package conf

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// Format identifies a value encoding.
type Format string

const (
	// FormatYAML parses YAML documents and plain scalars.
	FormatYAML Format = "yaml"
	// FormatJSON parses strict JSON.
	FormatJSON Format = "json"
	// FormatTOML parses TOML.
	FormatTOML Format = "toml"
	// FormatScalar parses one raw value into the target type without a
	// document parser. It is valid for etcd leaves only.
	FormatScalar Format = "scalar"
)

var (
	textUnmarshalerType = reflect.TypeFor[encoding.TextUnmarshaler]()
	durationType        = reflect.TypeFor[time.Duration]()
)

// Validate reports whether f is a known format.
func (f Format) Validate() error {
	switch f {
	case FormatYAML, FormatJSON, FormatTOML, FormatScalar:
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrInvalidFormat, f)
	}
}

// parseValue decodes raw into a freshly allocated value of t using format.
// Error text never includes raw.
func parseValue(raw string, t reflect.Type, format Format) (any, error) {
	if format == FormatScalar {
		return parseScalar(raw, t)
	}

	ptr := reflect.New(t)
	var err error
	switch format {
	case FormatJSON:
		err = json.Unmarshal([]byte(raw), ptr.Interface())
	case FormatTOML:
		err = unmarshalTOML([]byte(raw), ptr.Interface())
		if err != nil {
			if v, werr := parseTOMLScalar(raw, t); werr == nil {
				return v, nil
			}
		}
	default:
		err = yaml.Unmarshal([]byte(raw), ptr.Interface())
	}
	if err != nil {
		return nil, fmt.Errorf("%w: cannot decode %s into %s", ErrParse, format, t)
	}
	return ptr.Elem().Interface(), nil
}

func unmarshalTOML(data []byte, v any) error {
	dec := toml.NewDecoder(bytes.NewReader(data))
	dec.EnableUnmarshalerInterface()
	return dec.Decode(v)
}

// parseTOMLScalar decodes one bare TOML value into t by wrapping it in a
// single-key document.
func parseTOMLScalar(raw string, t reflect.Type) (any, error) {
	mapType := reflect.MapOf(reflect.TypeFor[string](), t)
	mp := reflect.New(mapType)
	wrapped := "v = " + raw
	if !strings.HasSuffix(wrapped, "\n") {
		wrapped += "\n"
	}
	if err := unmarshalTOML([]byte(wrapped), mp.Interface()); err != nil {
		return nil, err
	}
	v := mp.Elem().MapIndex(reflect.ValueOf("v"))
	if !v.IsValid() {
		return nil, ErrParse
	}
	return v.Interface(), nil
}

// parseScalar decodes one bare value into t.
func parseScalar(raw string, t reflect.Type) (any, error) {
	ptr := reflect.New(t)
	if ptr.Type().Implements(textUnmarshalerType) {
		if err := ptr.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(raw)); err != nil {
			return nil, fmt.Errorf("%w: cannot decode scalar into %s", ErrParse, t)
		}
		return ptr.Elem().Interface(), nil
	}

	if t == durationType {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: cannot decode scalar into %s", ErrParse, t)
		}
		return d, nil
	}

	switch t.Kind() {
	case reflect.String:
		ptr.Elem().SetString(raw)
	case reflect.Bool:
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: cannot decode scalar into %s", ErrParse, t)
		}
		ptr.Elem().SetBool(v)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := strconv.ParseInt(raw, 10, t.Bits())
		if err != nil {
			return nil, fmt.Errorf("%w: cannot decode scalar into %s", ErrParse, t)
		}
		ptr.Elem().SetInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := strconv.ParseUint(raw, 10, t.Bits())
		if err != nil {
			return nil, fmt.Errorf("%w: cannot decode scalar into %s", ErrParse, t)
		}
		ptr.Elem().SetUint(v)
	case reflect.Float32, reflect.Float64:
		v, err := strconv.ParseFloat(raw, t.Bits())
		if err != nil {
			return nil, fmt.Errorf("%w: cannot decode scalar into %s", ErrParse, t)
		}
		ptr.Elem().SetFloat(v)
	default:
		return nil, fmt.Errorf("%w: scalar cannot decode into %s", ErrInvalidType, t)
	}
	return ptr.Elem().Interface(), nil
}
