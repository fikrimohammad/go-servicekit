package eval

import (
	"context"
	"errors"
	"testing"

	"github.com/fikrimohammad/go-servicekit/confloader"
	"github.com/fikrimohammad/go-servicekit/confloader/client"
)

// appConfig is the shape loadDebug must load.
type appConfig struct {
	Debug confloader.Getter[bool] `conf:"folder=settings,key=debug"`
}

// fake is an in-memory backend. Fetch returns ErrNotFound for a missing key.
type fake struct {
	m map[string]string
}

func (f fake) Fetch(_ context.Context, folder, key string) (client.Fetched, error) {
	v, ok := f.m[folder+"/"+key]
	if !ok {
		return client.Fetched{}, client.ErrNotFound
	}
	return client.Fetched{Value: v, Revision: "1"}, nil
}

func (fake) Close() error { return nil }

// loadDebug loads appConfig through c and returns Debug.
//
// TODO: build the loader with confloader.New[appConfig] and
// confloader.WithClient(c), read Data().Debug.Get(ctx), and Stop the loader.
func loadDebug(_ context.Context, _ client.Client) (bool, error) {
	_ = confloader.Config{}
	return false, errors.New("not implemented")
}

func TestLoadDebug(t *testing.T) {
	c := fake{m: map[string]string{"settings/debug": "true"}}

	got, err := loadDebug(context.Background(), c)
	if err != nil {
		t.Fatalf("loadDebug: %v", err)
	}
	if !got {
		t.Error("debug = false, want true")
	}
}
