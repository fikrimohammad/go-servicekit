package conf

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	p := &fakeProvider{values: validValues()}
	loader, err := New[testConfig](context.Background(), p)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { _ = loader.Close() })

	wantKeys := []string{"checkouts/ttl", "checkouts/max_retry", "toggle_switch"}
	if got := p.recordedKeys(); !slices.Equal(got, wantKeys) {
		t.Errorf("Load keys = %v, want %v", got, wantKeys)
	}

	if got := loader.Data().Checkout.Timeout.GetWithDefault(0); got != 30*time.Second {
		t.Errorf("Checkout.Timeout = %v, want 30s", got)
	}
	if got := loader.Data().Checkout.MaxRetries.GetWithDefault(0); got != 3 {
		t.Errorf("Checkout.MaxRetries = %d, want 3", got)
	}
	if got := loader.Data().KillSwitch.GetWithDefault(false); !got {
		t.Error("KillSwitch = false, want true")
	}
	if got := loader.State(); got != StateLive {
		t.Errorf("State() = %q, want %q", got, StateLive)
	}
}

func TestNewMissingKey(t *testing.T) {
	p := &fakeProvider{values: map[string]string{"checkouts/ttl": "30s"}}
	_, err := New[testConfig](context.Background(), p)
	if !errors.Is(err, ErrMissingKey) {
		t.Fatalf("New() error = %v, want ErrMissingKey", err)
	}
	if got := p.closeCount(); got != 1 {
		t.Errorf("provider Close calls = %d, want 1", got)
	}
}

func TestNewParseFailure(t *testing.T) {
	values := validValues()
	values["checkouts/ttl"] = "not-a-duration"
	p := &fakeProvider{values: values}
	_, err := New[testConfig](context.Background(), p)
	if !errors.Is(err, ErrParse) {
		t.Fatalf("New() error = %v, want ErrParse", err)
	}
}

func TestNewNilProvider(t *testing.T) {
	_, err := New[testConfig](context.Background(), nil)
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("New() error = %v, want ErrInvalidConfig", err)
	}
}

func TestNewNonStruct(t *testing.T) {
	_, err := New[int](context.Background(), &fakeProvider{})
	if !errors.Is(err, ErrInvalidType) {
		t.Fatalf("New() error = %v, want ErrInvalidType", err)
	}
}

func TestNewLoadErrorClosesProvider(t *testing.T) {
	loadErr := errors.New("load failed")
	p := &fakeProvider{loadErr: loadErr}
	_, err := New[testConfig](context.Background(), p)
	if !errors.Is(err, loadErr) {
		t.Fatalf("New() error = %v, want %v", err, loadErr)
	}
	if got := p.closeCount(); got != 1 {
		t.Errorf("provider Close calls = %d, want 1", got)
	}
}

func TestNewFileMode(t *testing.T) {
	p := &fakeFormatter{
		fakeProvider: &fakeProvider{values: map[string]string{"checkouts/ttl": "30s"}},
		format:       FormatYAML,
	}
	loader, err := New[fileConfig](context.Background(), p)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { _ = loader.Close() })

	if got := loader.Data().Checkout.Timeout.GetWithDefault(0); got != 30*time.Second {
		t.Errorf("Checkout.Timeout = %v, want 30s", got)
	}
}

func TestNewFileModeRejectsFormatTag(t *testing.T) {
	p := &fakeFormatter{
		fakeProvider: &fakeProvider{values: validValues()},
		format:       FormatYAML,
	}
	_, err := New[testConfig](context.Background(), p)
	if !errors.Is(err, ErrInvalidTag) {
		t.Fatalf("New() error = %v, want ErrInvalidTag", err)
	}
}

func TestNewRemoteRequiresFormat(t *testing.T) {
	p := &fakeProvider{values: map[string]string{"checkouts/ttl": "30s"}}
	_, err := New[fileConfig](context.Background(), p)
	if !errors.Is(err, ErrMissingFormat) {
		t.Fatalf("New() error = %v, want ErrMissingFormat", err)
	}
}

func TestClose(t *testing.T) {
	p := &fakeProvider{values: validValues()}
	loader, err := New[testConfig](context.Background(), p)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := loader.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if got := loader.State(); got != StateClosed {
		t.Errorf("State() = %q, want %q", got, StateClosed)
	}
	if _, err := loader.Data().KillSwitch.Get(); !errors.Is(err, ErrClosed) {
		t.Errorf("Get() error = %v, want ErrClosed", err)
	}
	if got := loader.Data().KillSwitch.GetWithDefault(false); got {
		t.Error("GetWithDefault() = true, want false")
	}
	if got := p.closeCount(); got != 1 {
		t.Errorf("provider Close calls = %d, want 1", got)
	}

	if err := loader.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	if got := p.closeCount(); got != 1 {
		t.Errorf("provider Close calls after second Close = %d, want 1", got)
	}
}
