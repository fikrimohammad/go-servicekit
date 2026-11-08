package etcd

import (
	"context"
	"errors"
	"testing"

	"github.com/fikrimohammad/go-servicekit/conf"
)

func TestKeyFromPrefix(t *testing.T) {
	if got := keyFromPrefix("app/", "app/checkouts/ttl"); got != "checkouts/ttl" {
		t.Errorf("keyFromPrefix() = %q, want checkouts/ttl", got)
	}
}

func TestSource(t *testing.T) {
	if got := New(Config{}).Source(); got != "etcd" {
		t.Errorf("Source() = %q, want etcd", got)
	}
}

func TestLoadInvalidConfig(t *testing.T) {
	p := New(Config{})
	if _, _, err := p.Load(context.Background(), nil); !errors.Is(err, conf.ErrInvalidConfig) {
		t.Errorf("Load() error = %v, want ErrInvalidConfig", err)
	}
}

func TestWatchBeforeLoad(t *testing.T) {
	p := New(Config{Endpoint: "localhost:2379", Namespace: "app"})
	if _, err := p.Watch(context.Background(), 0); !errors.Is(err, conf.ErrInvalidConfig) {
		t.Errorf("Watch() error = %v, want ErrInvalidConfig", err)
	}
}
