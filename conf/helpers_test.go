package conf

import (
	"context"
	"reflect"
	"sync"
	"time"
)

type testConfig struct {
	Checkout struct {
		Timeout    Getter[time.Duration] `conf:"name=ttl,format=scalar"`
		MaxRetries Getter[int]           `conf:"name=max_retry,format=scalar"`
	} `conf:"folder=checkouts"`
	KillSwitch Getter[bool] `conf:"name=toggle_switch,format=scalar"`
}

type fileConfig struct {
	Checkout struct {
		Timeout Getter[time.Duration] `conf:"name=ttl"`
	} `conf:"folder=checkouts"`
}

type fakeProvider struct {
	mu       sync.Mutex
	values   map[string]string
	keys     []string
	loadErr  error
	closeErr error
	rev      int64
	source   string
	closed   int
}

func (f *fakeProvider) Source() string {
	if f.source == "" {
		return "fake"
	}
	return f.source
}

func (f *fakeProvider) Load(_ context.Context, keys []string) (map[string]string, int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.keys = append([]string(nil), keys...)
	if f.loadErr != nil {
		return nil, 0, f.loadErr
	}
	return f.values, f.rev, nil
}

func (f *fakeProvider) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed++
	return f.closeErr
}

func (f *fakeProvider) recordedKeys() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.keys...)
}

func (f *fakeProvider) closeCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closed
}

type fakeFormatter struct {
	*fakeProvider
	format Format
}

func (f *fakeFormatter) Parse(raw string, t reflect.Type) (any, error) {
	return parseValue(raw, t, f.format)
}

type fakeWatcher struct {
	*fakeProvider
	ch       chan Event
	watchErr error

	mu       sync.Mutex
	rev      int64
	canceled bool
}

func (f *fakeWatcher) Watch(ctx context.Context, rev int64) (<-chan Event, error) {
	if f.watchErr != nil {
		return nil, f.watchErr
	}
	f.mu.Lock()
	f.rev = rev
	f.mu.Unlock()
	go func() {
		<-ctx.Done()
		f.mu.Lock()
		f.canceled = true
		f.mu.Unlock()
	}()
	return f.ch, nil
}

func (f *fakeWatcher) watchRev() int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.rev
}

func (f *fakeWatcher) isCanceled() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.canceled
}

func validValues() map[string]string {
	return map[string]string{
		"checkouts/ttl":       "30s",
		"checkouts/max_retry": "3",
		"toggle_switch":       "true",
	}
}
