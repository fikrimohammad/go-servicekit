package file

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/fikrimohammad/go-servicekit/conf"
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// Provider reads one config file.
type Provider struct {
	cfg Config
}

// New builds a file provider.
func New(cfg Config) *Provider { return &Provider{cfg: cfg} }

// Source returns "file".
func (p *Provider) Source() string { return "file" }

// Parse decodes a raw value with the configured document format.
func (p *Provider) Parse(raw string, t reflect.Type) (any, error) {
	switch p.cfg.Format {
	case conf.FormatJSON:
		return decodeJSON(raw, t)
	case conf.FormatTOML:
		return decodeTOML(raw, t)
	default:
		return decodeYAML(raw, t)
	}
}

// Load reads the file once and returns the requested keys in the document
// format.
func (p *Provider) Load(ctx context.Context, keys []string) (map[string]string, int64, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	cfg := p.cfg.SetDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, 0, err
	}

	data, err := os.ReadFile(cfg.Path)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %v", conf.ErrInvalidConfig, err)
	}

	switch cfg.Format {
	case conf.FormatJSON:
		return loadJSON(data, keys)
	case conf.FormatTOML:
		return loadTOML(data, keys)
	default:
		return loadYAML(data, keys)
	}
}

// Close is a no-op.
func (p *Provider) Close() error { return nil }

var (
	_ conf.Provider  = (*Provider)(nil)
	_ conf.Formatter = (*Provider)(nil)
)

func decodeYAML(raw string, t reflect.Type) (any, error) {
	ptr := reflect.New(t)
	if err := yaml.Unmarshal([]byte(raw), ptr.Interface()); err != nil {
		return nil, fmt.Errorf("%w: cannot decode YAML into %s", conf.ErrParse, t)
	}
	return ptr.Elem().Interface(), nil
}

func decodeJSON(raw string, t reflect.Type) (any, error) {
	ptr := reflect.New(t)
	if err := json.Unmarshal([]byte(raw), ptr.Interface()); err != nil {
		return nil, fmt.Errorf("%w: cannot decode JSON into %s", conf.ErrParse, t)
	}
	return ptr.Elem().Interface(), nil
}

func decodeTOML(raw string, t reflect.Type) (any, error) {
	ptr := reflect.New(t)
	if err := unmarshalTOML([]byte(raw), ptr.Interface()); err == nil {
		return ptr.Elem().Interface(), nil
	}

	mapType := reflect.MapOf(reflect.TypeFor[string](), t)
	mp := reflect.New(mapType)
	if err := unmarshalTOML([]byte(raw), mp.Interface()); err == nil {
		v := mp.Elem().MapIndex(reflect.ValueOf("v"))
		if v.IsValid() {
			return v.Interface(), nil
		}
	}
	return nil, fmt.Errorf("%w: cannot decode TOML into %s", conf.ErrParse, t)
}

func unmarshalTOML(data []byte, v any) error {
	dec := toml.NewDecoder(bytes.NewReader(data))
	dec.EnableUnmarshalerInterface()
	return dec.Decode(v)
}

func loadYAML(data []byte, keys []string) (map[string]string, int64, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, 0, fmt.Errorf("%w: cannot parse YAML document", conf.ErrParse)
	}

	requested, prefixes := keyIndex(keys)
	values := make(map[string]string)
	var unknown []string
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		if err := walkYAML(root.Content[0], "", requested, prefixes, values, &unknown); err != nil {
			return nil, 0, err
		}
	}
	if err := unknownError(unknown); err != nil {
		return nil, 0, err
	}
	return values, 0, nil
}

func walkYAML(node *yaml.Node, path string, requested, prefixes map[string]struct{}, values map[string]string, unknown *[]string) error {
	if _, ok := requested[path]; ok {
		raw, err := yaml.Marshal(node)
		if err != nil {
			return fmt.Errorf("%w: cannot encode YAML leaf %s", conf.ErrParse, path)
		}
		values[path] = string(raw)
		return nil
	}
	if node.Kind != yaml.MappingNode {
		*unknown = append(*unknown, path)
		return nil
	}

	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i].Value
		childPath := key
		if path != "" {
			childPath = path + "/" + key
		}
		child := node.Content[i+1]

		if _, ok := requested[childPath]; ok {
			raw, err := yaml.Marshal(child)
			if err != nil {
				return fmt.Errorf("%w: cannot encode YAML leaf %s", conf.ErrParse, childPath)
			}
			values[childPath] = string(raw)
			continue
		}
		if _, ok := prefixes[childPath]; ok {
			if err := walkYAML(child, childPath, requested, prefixes, values, unknown); err != nil {
				return err
			}
			continue
		}
		*unknown = append(*unknown, childPath)
	}
	return nil
}

func loadJSON(data []byte, keys []string) (map[string]string, int64, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var root any
	if err := dec.Decode(&root); err != nil {
		return nil, 0, fmt.Errorf("%w: cannot parse JSON document", conf.ErrParse)
	}

	requested, prefixes := keyIndex(keys)
	values := make(map[string]string)
	var unknown []string
	if err := walkJSON(root, "", requested, prefixes, values, &unknown); err != nil {
		return nil, 0, err
	}
	if err := unknownError(unknown); err != nil {
		return nil, 0, err
	}
	return values, 0, nil
}

func walkJSON(value any, path string, requested, prefixes map[string]struct{}, values map[string]string, unknown *[]string) error {
	if _, ok := requested[path]; ok {
		raw, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("%w: cannot encode JSON leaf %s", conf.ErrParse, path)
		}
		values[path] = string(raw)
		return nil
	}

	m, ok := value.(map[string]any)
	if !ok {
		*unknown = append(*unknown, path)
		return nil
	}
	for key, child := range m {
		childPath := key
		if path != "" {
			childPath = path + "/" + key
		}
		if _, ok := requested[childPath]; ok {
			raw, err := json.Marshal(child)
			if err != nil {
				return fmt.Errorf("%w: cannot encode JSON leaf %s", conf.ErrParse, childPath)
			}
			values[childPath] = string(raw)
			continue
		}
		if _, ok := prefixes[childPath]; ok {
			if err := walkJSON(child, childPath, requested, prefixes, values, unknown); err != nil {
				return err
			}
			continue
		}
		*unknown = append(*unknown, childPath)
	}
	return nil
}

func loadTOML(data []byte, keys []string) (map[string]string, int64, error) {
	var root map[string]any
	if err := toml.Unmarshal(data, &root); err != nil {
		return nil, 0, fmt.Errorf("%w: cannot parse TOML document", conf.ErrParse)
	}

	requested, prefixes := keyIndex(keys)
	values := make(map[string]string)
	var unknown []string
	if err := walkTOML(root, "", requested, prefixes, values, &unknown); err != nil {
		return nil, 0, err
	}
	if err := unknownError(unknown); err != nil {
		return nil, 0, err
	}
	return values, 0, nil
}

func walkTOML(value any, path string, requested, prefixes map[string]struct{}, values map[string]string, unknown *[]string) error {
	if _, ok := requested[path]; ok {
		raw, err := tomlLeaf(value)
		if err != nil {
			return fmt.Errorf("%w: cannot encode TOML leaf %s", conf.ErrParse, path)
		}
		values[path] = raw
		return nil
	}

	m, ok := value.(map[string]any)
	if !ok {
		*unknown = append(*unknown, path)
		return nil
	}
	for key, child := range m {
		childPath := key
		if path != "" {
			childPath = path + "/" + key
		}
		if _, ok := requested[childPath]; ok {
			raw, err := tomlLeaf(child)
			if err != nil {
				return fmt.Errorf("%w: cannot encode TOML leaf %s", conf.ErrParse, childPath)
			}
			values[childPath] = raw
			continue
		}
		if _, ok := prefixes[childPath]; ok {
			if err := walkTOML(child, childPath, requested, prefixes, values, unknown); err != nil {
				return err
			}
			continue
		}
		*unknown = append(*unknown, childPath)
	}
	return nil
}

func tomlLeaf(value any) (string, error) {
	if _, ok := value.(map[string]any); ok {
		b, err := toml.Marshal(value)
		return string(b), err
	}
	b, err := toml.Marshal(map[string]any{"v": value})
	return string(b), err
}

func keyIndex(keys []string) (requested, prefixes map[string]struct{}) {
	requested = make(map[string]struct{}, len(keys))
	prefixes = make(map[string]struct{})
	for _, key := range keys {
		requested[key] = struct{}{}
		parts := strings.Split(key, "/")
		for i := 1; i < len(parts); i++ {
			prefixes[strings.Join(parts[:i], "/")] = struct{}{}
		}
	}
	return requested, prefixes
}

func unknownError(unknown []string) error {
	if len(unknown) == 0 {
		return nil
	}
	return fmt.Errorf("%w: %s", conf.ErrUnknownKey, strings.Join(unknown, ", "))
}
