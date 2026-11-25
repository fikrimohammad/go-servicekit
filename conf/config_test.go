package conf

import (
	"errors"
	"testing"
	"time"
)

func TestFileConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     FileConfig
		wantErr error
	}{
		{name: "yaml", cfg: FileConfig{Path: "app.yaml", Format: FormatYAML}},
		{name: "json", cfg: FileConfig{Path: "app.json", Format: FormatJSON}},
		{name: "toml", cfg: FileConfig{Path: "app.toml", Format: FormatTOML}},
		{name: "empty path", cfg: FileConfig{Format: FormatYAML}, wantErr: ErrInvalidConfig},
		{name: "empty format", cfg: FileConfig{Path: "app.yaml"}, wantErr: ErrInvalidFormat},
		{name: "scalar format", cfg: FileConfig{Path: "app.yaml", Format: FormatScalar}, wantErr: ErrInvalidFormat},
		{name: "unknown format", cfg: FileConfig{Path: "app.yaml", Format: "xml"}, wantErr: ErrInvalidFormat},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.Validate(); !errors.Is(err, tt.wantErr) {
				t.Errorf("FileConfig.Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestFileConfigSetDefaults(t *testing.T) {
	cfg := FileConfig{Path: "app.yaml", Format: FormatYAML}
	if got := cfg.SetDefaults(); got != cfg {
		t.Errorf("FileConfig.SetDefaults() = %+v, want %+v", got, cfg)
	}
}

func TestEtcdConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     EtcdConfig
		wantErr error
	}{
		{name: "valid", cfg: EtcdConfig{Endpoint: "localhost:2379", Namespace: "app"}},
		{name: "auth", cfg: EtcdConfig{Endpoint: "localhost:2379", Namespace: "app", Username: "user", Password: "pass"}},
		{name: "empty endpoint", cfg: EtcdConfig{Namespace: "app"}, wantErr: ErrInvalidConfig},
		{name: "no port", cfg: EtcdConfig{Endpoint: "localhost", Namespace: "app"}, wantErr: ErrInvalidConfig},
		{name: "empty namespace", cfg: EtcdConfig{Endpoint: "localhost:2379"}, wantErr: ErrInvalidConfig},
		{name: "leading slash", cfg: EtcdConfig{Endpoint: "localhost:2379", Namespace: "/app"}, wantErr: ErrInvalidConfig},
		{name: "trailing slash", cfg: EtcdConfig{Endpoint: "localhost:2379", Namespace: "app/"}, wantErr: ErrInvalidConfig},
		{name: "double slash", cfg: EtcdConfig{Endpoint: "localhost:2379", Namespace: "a//b"}, wantErr: ErrInvalidConfig},
		{name: "username only", cfg: EtcdConfig{Endpoint: "localhost:2379", Namespace: "app", Username: "user"}, wantErr: ErrInvalidConfig},
		{name: "password only", cfg: EtcdConfig{Endpoint: "localhost:2379", Namespace: "app", Password: "pass"}, wantErr: ErrInvalidConfig},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.Validate(); !errors.Is(err, tt.wantErr) {
				t.Errorf("EtcdConfig.Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestEtcdConfigSetDefaults(t *testing.T) {
	cfg := EtcdConfig{Endpoint: "localhost:2379", Namespace: "app"}
	if got := cfg.SetDefaults().DialTimeout; got != 5*time.Second {
		t.Errorf("EtcdConfig.SetDefaults().DialTimeout = %v, want 5s", got)
	}

	cfg.DialTimeout = time.Second
	if got := cfg.SetDefaults().DialTimeout; got != time.Second {
		t.Errorf("EtcdConfig.SetDefaults().DialTimeout = %v, want 1s", got)
	}
}
