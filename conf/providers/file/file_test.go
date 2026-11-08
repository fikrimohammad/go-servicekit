package file

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/fikrimohammad/go-servicekit/conf"
	"gopkg.in/yaml.v3"
)

type rules struct {
	Max int `yaml:"max" json:"max" toml:"max"`
}

func TestLoadYAML(t *testing.T) {
	data := "checkouts:\n  ttl: 30s\n  max_retry: 3\ntoggle_switch: true\n"
	keys := []string{"checkouts/ttl", "checkouts/max_retry", "toggle_switch"}
	values, p := loadValues(t, conf.FormatYAML, "app.yaml", data, keys)

	got, err := p.Parse(values["checkouts/ttl"], reflect.TypeFor[time.Duration]())
	if err != nil || got != 30*time.Second {
		t.Errorf("Parse(ttl) = %v, %v, want 30s", got, err)
	}
	got, err = p.Parse(values["checkouts/max_retry"], reflect.TypeFor[int]())
	if err != nil || got != 3 {
		t.Errorf("Parse(max_retry) = %v, %v, want 3", got, err)
	}
	got, err = p.Parse(values["toggle_switch"], reflect.TypeFor[bool]())
	if err != nil || got != true {
		t.Errorf("Parse(toggle_switch) = %v, %v, want true", got, err)
	}
}

func TestLoadJSON(t *testing.T) {
	data := `{"checkouts":{"ttl":30,"account_id":9007199254740993},"toggle_switch":true}`
	keys := []string{"checkouts/ttl", "checkouts/account_id", "toggle_switch"}
	values, p := loadValues(t, conf.FormatJSON, "app.json", data, keys)

	if got := strings.TrimSpace(values["checkouts/account_id"]); got != "9007199254740993" {
		t.Errorf("raw account_id = %q, want 9007199254740993", got)
	}
	got, err := p.Parse(values["checkouts/account_id"], reflect.TypeFor[int64]())
	if err != nil || got != int64(9007199254740993) {
		t.Errorf("Parse(account_id) = %v, %v, want 9007199254740993", got, err)
	}
	got, err = p.Parse(values["checkouts/ttl"], reflect.TypeFor[int]())
	if err != nil || got != 30 {
		t.Errorf("Parse(ttl) = %v, %v, want 30", got, err)
	}
}

func TestLoadTOML(t *testing.T) {
	data := "[checkouts]\nttl = 30\nmax_retry = 3\n\n[toggle]\nenabled = true\n"
	keys := []string{"checkouts/ttl", "checkouts/max_retry", "toggle/enabled"}
	values, p := loadValues(t, conf.FormatTOML, "app.toml", data, keys)

	got, err := p.Parse(values["checkouts/ttl"], reflect.TypeFor[int]())
	if err != nil || got != 30 {
		t.Errorf("Parse(ttl) = %v, %v, want 30", got, err)
	}
	got, err = p.Parse(values["toggle/enabled"], reflect.TypeFor[bool]())
	if err != nil || got != true {
		t.Errorf("Parse(enabled) = %v, %v, want true", got, err)
	}
}

func TestLoadTOMLTableLeaf(t *testing.T) {
	data := "[rules]\nmax = 3\n"
	values, p := loadValues(t, conf.FormatTOML, "app.toml", data, []string{"rules"})

	got, err := p.Parse(values["rules"], reflect.TypeFor[rules]())
	if err != nil {
		t.Fatalf("Parse(rules) error = %v", err)
	}
	if got.(rules).Max != 3 {
		t.Errorf("Parse(rules) = %+v, want Max 3", got)
	}
}

func TestLoadTOMLArrayLeaf(t *testing.T) {
	data := "[limits]\nvalues = [1, 2]\n"
	values, p := loadValues(t, conf.FormatTOML, "app.toml", data, []string{"limits/values"})

	got, err := p.Parse(values["limits/values"], reflect.TypeFor[[]int]())
	if err != nil {
		t.Fatalf("Parse(values) error = %v", err)
	}
	if want := []int{1, 2}; !slices.Equal(got.([]int), want) {
		t.Errorf("Parse(values) = %v, want %v", got, want)
	}
}

func TestLoadYAMLStructLeaf(t *testing.T) {
	data := "rules:\n  max: 3\n"
	values, p := loadValues(t, conf.FormatYAML, "app.yaml", data, []string{"rules"})

	got, err := p.Parse(values["rules"], reflect.TypeFor[rules]())
	if err != nil {
		t.Fatalf("Parse(rules) error = %v", err)
	}
	if got.(rules).Max != 3 {
		t.Errorf("Parse(rules) = %+v, want Max 3", got)
	}
}

func TestLoadUnknownKey(t *testing.T) {
	tests := []struct {
		name   string
		format conf.Format
		file   string
		data   string
		keys   []string
	}{
		{
			name:   "yaml",
			format: conf.FormatYAML,
			file:   "app.yaml",
			data:   "checkouts:\n  ttl: 30s\nextra: 1\n",
			keys:   []string{"checkouts/ttl"},
		},
		{
			name:   "json",
			format: conf.FormatJSON,
			file:   "app.json",
			data:   `{"checkouts":{"ttl":30},"extra":1}`,
			keys:   []string{"checkouts/ttl"},
		},
		{
			name:   "toml",
			format: conf.FormatTOML,
			file:   "app.toml",
			data:   "[checkouts]\nttl = 30\n[extra]\nx = 1\n",
			keys:   []string{"checkouts/ttl"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTemp(t, tt.file, tt.data)
			p := New(Config{Path: path, Format: tt.format})
			if _, _, err := p.Load(context.Background(), tt.keys); !errors.Is(err, conf.ErrUnknownKey) {
				t.Errorf("Load() error = %v, want ErrUnknownKey", err)
			}
		})
	}
}

func TestLoadMissingKey(t *testing.T) {
	data := "checkouts:\n  ttl: 30s\n"
	values, _ := loadValues(t, conf.FormatYAML, "app.yaml", data, []string{"checkouts/ttl", "checkouts/max_retry"})
	if _, ok := values["checkouts/max_retry"]; ok {
		t.Error("missing key unexpectedly present")
	}
}

func TestLoadMalformed(t *testing.T) {
	tests := []struct {
		name   string
		format conf.Format
		file   string
		data   string
	}{
		{name: "yaml", format: conf.FormatYAML, file: "app.yaml", data: "not: [valid"},
		{name: "json", format: conf.FormatJSON, file: "app.json", data: "{"},
		{name: "toml", format: conf.FormatTOML, file: "app.toml", data: "x ="},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTemp(t, tt.file, tt.data)
			p := New(Config{Path: path, Format: tt.format})
			if _, _, err := p.Load(context.Background(), nil); !errors.Is(err, conf.ErrParse) {
				t.Errorf("Load() error = %v, want ErrParse", err)
			}
		})
	}
}

func TestLoadScalarFormat(t *testing.T) {
	p := New(Config{Path: "app.yaml", Format: conf.FormatScalar})
	if _, _, err := p.Load(context.Background(), nil); !errors.Is(err, conf.ErrInvalidFormat) {
		t.Errorf("Load() error = %v, want ErrInvalidFormat", err)
	}
}

func TestLoadMissingFile(t *testing.T) {
	p := New(Config{Path: filepath.Join(t.TempDir(), "missing.yaml"), Format: conf.FormatYAML})
	if _, _, err := p.Load(context.Background(), nil); !errors.Is(err, conf.ErrInvalidConfig) {
		t.Errorf("Load() error = %v, want ErrInvalidConfig", err)
	}
}

func TestLoadContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := New(Config{Path: "app.yaml", Format: conf.FormatYAML})
	if _, _, err := p.Load(ctx, nil); !errors.Is(err, context.Canceled) {
		t.Errorf("Load() error = %v, want context.Canceled", err)
	}
}

func TestSource(t *testing.T) {
	if got := New(Config{}).Source(); got != "file" {
		t.Errorf("Source() = %q, want file", got)
	}
}

type jsonPrice struct {
	Called bool
	Raw    string
}

func (p *jsonPrice) UnmarshalJSON(b []byte) error {
	p.Called = true
	p.Raw = string(b)
	return nil
}

type yamlPrice struct {
	Called bool
	Tag    string
}

func (p *yamlPrice) UnmarshalYAML(n *yaml.Node) error {
	p.Called = true
	p.Tag = n.Tag
	return nil
}

type tomlPrice struct {
	Called bool
	Data   string
}

func (p *tomlPrice) UnmarshalTOML(data []byte) error {
	p.Called = true
	p.Data = string(data)
	return nil
}

func TestParseCustomUnmarshalers(t *testing.T) {
	t.Run("json", func(t *testing.T) {
		values, p := loadValues(t, conf.FormatJSON, "app.json", `{"price":{"value":100}}`, []string{"price"})
		got, err := p.Parse(values["price"], reflect.TypeFor[jsonPrice]())
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}
		if !got.(jsonPrice).Called {
			t.Error("UnmarshalJSON was not called")
		}
	})

	t.Run("yaml", func(t *testing.T) {
		values, p := loadValues(t, conf.FormatYAML, "app.yaml", "price:\n  value: 100\n", []string{"price"})
		got, err := p.Parse(values["price"], reflect.TypeFor[yamlPrice]())
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}
		if !got.(yamlPrice).Called {
			t.Error("UnmarshalYAML was not called")
		}
	})

	t.Run("toml", func(t *testing.T) {
		values, p := loadValues(t, conf.FormatTOML, "app.toml", "[price]\nvalue = 100\n", []string{"price"})
		got, err := p.Parse(values["price"], reflect.TypeFor[tomlPrice]())
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}
		if !got.(tomlPrice).Called {
			t.Error("UnmarshalTOML was not called")
		}
	})
}

func loadValues(t *testing.T, format conf.Format, name, data string, keys []string) (map[string]string, *Provider) {
	t.Helper()
	path := writeTemp(t, name, data)
	p := New(Config{Path: path, Format: format})
	values, _, err := p.Load(context.Background(), keys)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	return values, p
}

func writeTemp(t *testing.T, name, data string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
	return path
}
