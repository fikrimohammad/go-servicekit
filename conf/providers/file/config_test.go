package file

import (
	"errors"
	"testing"

	"github.com/fikrimohammad/go-servicekit/conf"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr error
	}{
		{name: "yaml", cfg: Config{Path: "app.yaml", Format: conf.FormatYAML}},
		{name: "json", cfg: Config{Path: "app.json", Format: conf.FormatJSON}},
		{name: "toml", cfg: Config{Path: "app.toml", Format: conf.FormatTOML}},
		{name: "empty path", cfg: Config{Format: conf.FormatYAML}, wantErr: conf.ErrInvalidConfig},
		{name: "empty format", cfg: Config{Path: "app.yaml"}, wantErr: conf.ErrInvalidFormat},
		{name: "scalar format", cfg: Config{Path: "app.yaml", Format: conf.FormatScalar}, wantErr: conf.ErrInvalidFormat},
		{name: "unknown format", cfg: Config{Path: "app.yaml", Format: "xml"}, wantErr: conf.ErrInvalidFormat},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.Validate(); !errors.Is(err, tt.wantErr) {
				t.Errorf("Config.Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigSetDefaults(t *testing.T) {
	cfg := Config{Path: "app.yaml", Format: conf.FormatYAML}
	if got := cfg.SetDefaults(); got != cfg {
		t.Errorf("Config.SetDefaults() = %+v, want %+v", got, cfg)
	}
}
