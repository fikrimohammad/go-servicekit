package conf

import (
	"errors"
	"testing"
	"time"
)

func TestFileSource(t *testing.T) {
	loader, err := New[fileConfig](t.Context(), File(FileConfig{
		Path:   "testdata/app.yaml",
		Format: FormatYAML,
	}))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { _ = loader.Close() })

	if got := loader.Data().Checkout.Timeout.GetWithDefault(0); got != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", got)
	}
}

func TestFileSourceReusable(t *testing.T) {
	source := File(FileConfig{Path: "testdata/app.yaml", Format: FormatYAML})
	for range 2 {
		loader, err := New[fileConfig](t.Context(), source)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		if err := loader.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}
}

func TestSourceZeroValue(t *testing.T) {
	_, err := New[testConfig](t.Context(), Source{})
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("New() error = %v, want ErrInvalidConfig", err)
	}
}

func TestFileSourceInvalidConfig(t *testing.T) {
	_, err := New[fileConfig](t.Context(), File(FileConfig{}))
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("New() error = %v, want ErrInvalidConfig", err)
	}
}

func TestEtcdSourceInvalidConfig(t *testing.T) {
	_, err := New[testConfig](t.Context(), Etcd(EtcdConfig{}))
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("New() error = %v, want ErrInvalidConfig", err)
	}
}
