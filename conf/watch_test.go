package conf

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestWatchPut(t *testing.T) {
	ch := make(chan Event, 1)
	loader, p := newWatchLoader(t, ch)

	ch <- Event{Key: "toggle_switch", Value: "false", Revision: 43}
	waitFor(t, func() bool {
		return !loader.Data().KillSwitch.GetWithDefault(true)
	})

	if got := p.watchRev(); got != 42 {
		t.Errorf("Watch revision = %d, want 42", got)
	}
	if got := loader.State(); got != StateLive {
		t.Errorf("State() = %q, want %q", got, StateLive)
	}
}

func TestWatchDelete(t *testing.T) {
	ch := make(chan Event, 1)
	loader, _ := newWatchLoader(t, ch)

	ch <- Event{Key: "toggle_switch", Deleted: true, Revision: 43}
	waitFor(t, func() bool {
		_, err := loader.Data().KillSwitch.Get()
		return errors.Is(err, ErrNotSet)
	})

	if got := loader.Data().KillSwitch.GetWithDefault(false); got {
		t.Error("GetWithDefault() = true, want false")
	}
	if got := loader.State(); got != StateLive {
		t.Errorf("State() = %q, want %q", got, StateLive)
	}
}

func TestWatchParseFailure(t *testing.T) {
	ch := make(chan Event, 2)
	loader, _ := newWatchLoader(t, ch)

	ch <- Event{Key: "checkouts/ttl", Value: "not-a-duration"}
	waitFor(t, func() bool {
		return loader.State() == StateDegraded
	})
	if got := loader.Data().Checkout.Timeout.GetWithDefault(0); got != 30*time.Second {
		t.Errorf("Timeout after bad value = %v, want 30s", got)
	}

	ch <- Event{Key: "checkouts/ttl", Value: "45s"}
	waitFor(t, func() bool {
		return loader.State() == StateLive
	})
	if got := loader.Data().Checkout.Timeout.GetWithDefault(0); got != 45*time.Second {
		t.Errorf("Timeout after recovery = %v, want 45s", got)
	}
}

func TestWatchReset(t *testing.T) {
	ch := make(chan Event, 1)
	loader, _ := newWatchLoader(t, ch)

	ch <- Event{Reset: map[string]string{
		"checkouts/ttl":       "45s",
		"checkouts/max_retry": "5",
	}}
	waitFor(t, func() bool {
		return loader.Data().Checkout.MaxRetries.GetWithDefault(0) == 5
	})

	if got := loader.Data().Checkout.Timeout.GetWithDefault(0); got != 45*time.Second {
		t.Errorf("Timeout = %v, want 45s", got)
	}
	if _, err := loader.Data().KillSwitch.Get(); !errors.Is(err, ErrNotSet) {
		t.Errorf("KillSwitch.Get() error = %v, want ErrNotSet", err)
	}
	if got := loader.State(); got != StateLive {
		t.Errorf("State() = %q, want %q", got, StateLive)
	}
}

func TestWatchResetKeepsLastGood(t *testing.T) {
	ch := make(chan Event, 1)
	loader, _ := newWatchLoader(t, ch)

	ch <- Event{Reset: map[string]string{
		"checkouts/ttl":       "not-a-duration",
		"checkouts/max_retry": "5",
	}}
	waitFor(t, func() bool {
		return loader.State() == StateDegraded
	})

	if got := loader.Data().Checkout.Timeout.GetWithDefault(0); got != 30*time.Second {
		t.Errorf("Timeout after bad reset = %v, want 30s", got)
	}
	if got := loader.Data().Checkout.MaxRetries.GetWithDefault(0); got != 5 {
		t.Errorf("MaxRetries after reset = %d, want 5", got)
	}
}

func TestWatchHeartbeat(t *testing.T) {
	ch := make(chan Event, 1)
	loader, _ := newWatchLoader(t, ch)

	ch <- Event{Err: errors.New("watch down")}
	waitFor(t, func() bool {
		return loader.State() == StateDegraded
	})

	ch <- Event{Heartbeat: true}
	waitFor(t, func() bool {
		return loader.State() == StateLive
	})
}

func TestWatchStartError(t *testing.T) {
	watchErr := errors.New("watch failed")
	p := &fakeWatcher{
		fakeProvider: &fakeProvider{values: validValues()},
		watchErr:     watchErr,
	}
	_, err := New[testConfig](context.Background(), p)
	if !errors.Is(err, watchErr) {
		t.Fatalf("New() error = %v, want %v", err, watchErr)
	}
	if got := p.closeCount(); got != 1 {
		t.Errorf("provider Close calls = %d, want 1", got)
	}
}

func TestWatchClose(t *testing.T) {
	ch := make(chan Event, 1)
	loader, p := newWatchLoader(t, ch)

	if err := loader.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	waitFor(t, p.isCanceled)
	if got := loader.State(); got != StateClosed {
		t.Errorf("State() = %q, want %q", got, StateClosed)
	}
	if _, err := loader.Data().KillSwitch.Get(); !errors.Is(err, ErrClosed) {
		t.Errorf("Get() error = %v, want ErrClosed", err)
	}
}

func TestWatchConcurrentGet(t *testing.T) {
	ch := make(chan Event, 64)
	loader, _ := newWatchLoader(t, ch)

	var wg sync.WaitGroup
	wg.Go(func() {
		for range 1000 {
			_ = loader.Data().KillSwitch.GetWithDefault(false)
			_ = loader.Data().Checkout.MaxRetries.GetWithDefault(0)
		}
	})

	for i := range 50 {
		value := "true"
		if i%2 == 0 {
			value = "false"
		}
		ch <- Event{Key: "toggle_switch", Value: value}
		ch <- Event{Key: "checkouts/max_retry", Value: "3"}
	}
	wg.Wait()
}

func newWatchLoader(t *testing.T, ch chan Event) (*Loader[testConfig], *fakeWatcher) {
	t.Helper()
	p := &fakeWatcher{
		fakeProvider: &fakeProvider{values: validValues(), rev: 42},
		ch:           ch,
	}
	loader, err := New[testConfig](context.Background(), p)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { _ = loader.Close() })
	return loader, p
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition not met before deadline")
}
