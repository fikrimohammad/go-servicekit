package eval

import (
	"context"
	"errors"
	"testing"

	"github.com/fikrimohammad/go-servicekit/conf"
)

// appConfig is the shape loadDebug must load.
type appConfig struct {
	Debug conf.Getter[bool] `conf:"name=debug,format=scalar"`
}

// fake is an in-memory provider. Load returns the requested keys.
type fake struct {
	m map[string]string
}

func (f fake) Source() string { return "fake" }

func (f fake) Load(_ context.Context, keys []string) (map[string]string, int64, error) {
	values := make(map[string]string, len(keys))
	for _, key := range keys {
		if v, ok := f.m[key]; ok {
			values[key] = v
		}
	}
	return values, 0, nil
}

func (fake) Close() error { return nil }

// loadDebug loads appConfig through p and returns Debug.
//
// TODO: build the loader with conf.New[appConfig](ctx, p), read
// Data().Debug.Get(), and Close the loader.
func loadDebug(_ context.Context, _ conf.Provider) (bool, error) {
	return false, errors.New("not implemented")
}

func TestLoadDebug(t *testing.T) {
	p := fake{m: map[string]string{"debug": "true"}}

	got, err := loadDebug(context.Background(), p)
	if err != nil {
		t.Fatalf("loadDebug: %v", err)
	}
	if !got {
		t.Error("debug = false, want true")
	}
}
