package etcd

import (
	"errors"
	"testing"
	"time"

	"github.com/fikrimohammad/go-servicekit/conf"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr error
	}{
		{name: "valid", cfg: Config{Endpoint: "localhost:2379", Namespace: "app"}},
		{name: "auth", cfg: Config{Endpoint: "localhost:2379", Namespace: "app", Username: "user", Password: "pass"}},
		{name: "empty endpoint", cfg: Config{Namespace: "app"}, wantErr: conf.ErrInvalidConfig},
		{name: "no port", cfg: Config{Endpoint: "localhost", Namespace: "app"}, wantErr: conf.ErrInvalidConfig},
		{name: "empty namespace", cfg: Config{Endpoint: "localhost:2379"}, wantErr: conf.ErrInvalidConfig},
		{name: "leading slash", cfg: Config{Endpoint: "localhost:2379", Namespace: "/app"}, wantErr: conf.ErrInvalidConfig},
		{name: "trailing slash", cfg: Config{Endpoint: "localhost:2379", Namespace: "app/"}, wantErr: conf.ErrInvalidConfig},
		{name: "double slash", cfg: Config{Endpoint: "localhost:2379", Namespace: "a//b"}, wantErr: conf.ErrInvalidConfig},
		{name: "username only", cfg: Config{Endpoint: "localhost:2379", Namespace: "app", Username: "user"}, wantErr: conf.ErrInvalidConfig},
		{name: "password only", cfg: Config{Endpoint: "localhost:2379", Namespace: "app", Password: "pass"}, wantErr: conf.ErrInvalidConfig},
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
	cfg := Config{Endpoint: "localhost:2379", Namespace: "app"}
	if got := cfg.SetDefaults().DialTimeout; got != 5*time.Second {
		t.Errorf("SetDefaults().DialTimeout = %v, want 5s", got)
	}

	cfg.DialTimeout = time.Second
	if got := cfg.SetDefaults().DialTimeout; got != time.Second {
		t.Errorf("SetDefaults().DialTimeout = %v, want 1s", got)
	}
}
