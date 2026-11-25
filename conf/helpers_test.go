package conf

import (
	"context"
	"reflect"
	"sync"
	"time"

	confcore "github.com/fikrimohammad/go-servicekit/conf/internal/core"
)

// Event is a test alias for the internal provider event.
type Event = confcore.Event

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

type pointerFileConfig struct {
	Ptr Getter[*int]
}

type namedInts []int

type namedSliceFileConfig struct {
	Values Getter[namedInts]
}

type textStringer interface {
	String() string
}

type stringerValue struct {
	V string
}

func (s stringerValue) String() string { return s.V }

type stringerFileConfig struct {
	Value Getter[textStringer]
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
	parse  func(raw string, t reflect.Type) (any, error)
}

func (f *fakeFormatter) Parse(raw string, t reflect.Type) (any, error) {
	if f.parse != nil {
		return f.parse(raw, t)
	}
	return confcore.ParseValue(raw, t, confcore.Format(f.format))
}

type fakeWatcher struct {
	*fakeProvider

	ch       chan confcore.Event
	watchErr error

	mu       sync.Mutex
	rev      int64
	canceled bool
}

type fakeFormatterWatcher struct {
	*fakeWatcher

	format Format
	parse  func(raw string, t reflect.Type) (any, error)
}

func (f *fakeFormatterWatcher) Parse(raw string, t reflect.Type) (any, error) {
	if f.parse != nil {
		return f.parse(raw, t)
	}
	return confcore.ParseValue(raw, t, confcore.Format(f.format))
}

func (f *fakeWatcher) Watch(ctx context.Context, rev int64) (<-chan confcore.Event, error) {
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

// testSource wraps a fake provider for New.
func testSource(p confcore.Provider) Source {
	return Source{newProvider: func() confcore.Provider { return p }}
}

func validValues() map[string]string {
	return map[string]string{
		"checkouts/ttl":       "30s",
		"checkouts/max_retry": "3",
		"toggle_switch":       "true",
	}
}
