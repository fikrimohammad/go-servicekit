package conf

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

var (
	getterLeafType = reflect.TypeFor[getterLeaf]()
	segmentPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
)

// leaf is one Getter field resolved to a key.
type leaf struct {
	key    string
	typ    reflect.Type
	format Format
	index  []int
	field  string
}

// derive walks t and returns its leaves. ownsFormat is true when the provider
// implements Formatter; defaultFormat is then the provider's format.
func derive(t reflect.Type, ownsFormat bool) ([]leaf, error) {
	var leaves []leaf
	seen := make(map[string]string)
	if err := walk(t, nil, nil, ownsFormat, &leaves, seen); err != nil {
		return nil, err
	}
	return leaves, nil
}

func walk(t reflect.Type, path []string, index []int, ownsFormat bool, leaves *[]leaf, seen map[string]string) error {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		rawTag := field.Tag.Get("conf")

		if field.PkgPath != "" {
			if rawTag != "" && rawTag != "-" {
				return fmt.Errorf("%w: field %s is unexported", ErrInvalidType, field.Name)
			}
			continue
		}

		fieldType := field.Type
		fieldIndex := append(append([]int(nil), index...), i)
		parsed, err := parseTag(rawTag)
		if err != nil {
			return fmt.Errorf("%w: field %s: %v", ErrInvalidTag, field.Name, err)
		}

		if reflect.PointerTo(fieldType).Implements(getterLeafType) {
			if parsed.skip {
				return fmt.Errorf("%w: field %s: cannot skip a Getter", ErrInvalidTag, field.Name)
			}
			if parsed.folder != "" {
				return fmt.Errorf("%w: field %s: folder is only valid on a struct", ErrInvalidTag, field.Name)
			}

			segment := parsed.name
			if segment == "" {
				segment = snake(field.Name)
			}
			if err := validateSegment(segment); err != nil {
				return fmt.Errorf("%w: field %s: %v", ErrInvalidTag, field.Name, err)
			}

			key := strings.Join(append(path, segment), "/")
			if prev, ok := seen[key]; ok {
				return fmt.Errorf("%w: %q is used by %s and %s", ErrDuplicateKey, key, prev, field.Name)
			}
			seen[key] = field.Name

			var format Format
			if ownsFormat {
				if parsed.format != "" {
					return fmt.Errorf("%w: field %s: the provider owns the format", ErrInvalidTag, field.Name)
				}
			} else {
				if parsed.format == "" {
					return fmt.Errorf("%w: field %s", ErrMissingFormat, field.Name)
				}
				format = parsed.format
				if err := format.Validate(); err != nil {
					return fmt.Errorf("%w: field %s: %v", ErrInvalidFormat, field.Name, err)
				}
			}

			targetType := reflect.New(fieldType).Elem().Addr().Interface().(getterLeaf).target()
			*leaves = append(*leaves, leaf{
				key:    key,
				typ:    targetType,
				format: format,
				index:  fieldIndex,
				field:  field.Name,
			})
			continue
		}

		if fieldType.Kind() == reflect.Struct {
			if parsed.skip {
				continue
			}
			if parsed.name != "" {
				return fmt.Errorf("%w: field %s: name is only valid on a Getter", ErrInvalidTag, field.Name)
			}
			if parsed.format != "" {
				return fmt.Errorf("%w: field %s: format is only valid on a Getter", ErrInvalidTag, field.Name)
			}

			segment := parsed.folder
			if segment == "" {
				segment = snake(field.Name)
			}
			if err := validateSegment(segment); err != nil {
				return fmt.Errorf("%w: field %s: %v", ErrInvalidTag, field.Name, err)
			}

			if err := walk(fieldType, append(path, segment), fieldIndex, ownsFormat, leaves, seen); err != nil {
				return err
			}
			continue
		}

		if rawTag != "" && rawTag != "-" {
			if parsed.folder != "" {
				return fmt.Errorf("%w: field %s is not a struct", ErrInvalidType, field.Name)
			}
			return fmt.Errorf("%w: field %s: only a Getter or struct can carry a conf tag", ErrInvalidTag, field.Name)
		}
	}
	return nil
}

// confTag is the parsed conf struct tag.
type confTag struct {
	folder string
	name   string
	format Format
	skip   bool
}

func parseTag(raw string) (confTag, error) {
	var t confTag
	if raw == "" {
		return t, nil
	}
	if raw == "-" {
		t.skip = true
		return t, nil
	}

	for part := range strings.SplitSeq(raw, ",") {
		k, v, ok := strings.Cut(part, "=")
		if !ok || k == "" || v == "" {
			return confTag{}, fmt.Errorf("expected key=value in %q", part)
		}
		switch k {
		case "folder":
			if t.folder != "" {
				return confTag{}, fmt.Errorf("duplicate folder")
			}
			t.folder = v
		case "name":
			if t.name != "" {
				return confTag{}, fmt.Errorf("duplicate name")
			}
			t.name = v
		case "format":
			if t.format != "" {
				return confTag{}, fmt.Errorf("duplicate format")
			}
			t.format = Format(v)
		default:
			return confTag{}, fmt.Errorf("unknown key %q", k)
		}
	}
	return t, nil
}

func validateSegment(s string) error {
	if !segmentPattern.MatchString(s) {
		return fmt.Errorf("segment %q must match %s", s, segmentPattern)
	}
	return nil
}

// snake converts a Go field name to snake_case, keeping initialisms together.
func snake(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			if i > 0 && (isLower(s[i-1]) || isDigit(s[i-1]) || (i+1 < len(s) && isLower(s[i+1]))) {
				b.WriteByte('_')
			}
			b.WriteByte(c + ('a' - 'A'))
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

func isLower(c byte) bool { return c >= 'a' && c <= 'z' }

func isDigit(c byte) bool { return c >= '0' && c <= '9' }
