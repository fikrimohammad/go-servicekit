package etcd

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fikrimohammad/go-servicekit/conf/internal/core"
)

func TestCloseStopsWatch(t *testing.T) {
	p := New(Config{})
	canceled := make(chan struct{})
	done := make(chan struct{})
	p.watchCancel = func() { close(canceled) }
	p.watchDone = done
	go func() {
		<-canceled
		close(done)
	}()

	closed := make(chan error, 1)
	go func() { closed <- p.Close() }()
	select {
	case err := <-closed:
		if err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Close() did not cancel and wait for the watch")
	}

	if _, ok := <-canceled; ok {
		t.Error("watch context was not canceled")
	}
	if _, ok := <-done; ok {
		t.Error("watch goroutine was not waited for")
	}
	if err := p.Close(); err != nil {
		t.Errorf("second Close() error = %v", err)
	}
}

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
	if _, _, err := p.Load(context.Background(), nil); !errors.Is(err, core.ErrInvalidConfig) {
		t.Errorf("Load() error = %v, want ErrInvalidConfig", err)
	}
}

func TestWatchBeforeLoad(t *testing.T) {
	p := New(Config{Endpoint: "localhost:2379", Namespace: "app"})
	if _, err := p.Watch(context.Background(), 0); !errors.Is(err, core.ErrInvalidConfig) {
		t.Errorf("Watch() error = %v, want ErrInvalidConfig", err)
	}
}
