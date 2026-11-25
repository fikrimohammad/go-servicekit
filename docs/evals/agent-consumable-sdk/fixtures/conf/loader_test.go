package eval

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/fikrimohammad/go-servicekit/conf"
)

// appConfig is the shape loadDebug must load.
type appConfig struct {
	Debug conf.Getter[bool] `conf:"name=debug"`
}

// loadDebug loads appConfig from one YAML file and returns Debug.
//
// TODO: build the loader with conf.New[appConfig](ctx, conf.File(...)), read
// Data().Debug.Get(), and Close the loader.
func loadDebug(_ context.Context, _ string) (bool, error) {
	return false, errors.New("not implemented")
}

func TestLoadDebug(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.yaml")
	if err := os.WriteFile(path, []byte("debug: true\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}

	got, err := loadDebug(t.Context(), path)
	if err != nil {
		t.Fatalf("loadDebug: %v", err)
	}
	if !got {
		t.Error("debug = false, want true")
	}
}
