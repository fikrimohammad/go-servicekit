package conf

import (
	"context"
	"errors"
	"maps"
	"sync"
	"testing"

	"github.com/fikrimohammad/go-servicekit/observability/metrics"
	"github.com/fikrimohammad/go-servicekit/observability/tracer"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

type metricCall struct {
	kind       string
	instrument metrics.Instrument
	value      float64
	attrs      map[string]any
}

type recordMetrics struct {
	mu    sync.Mutex
	calls []metricCall
}

func (r *recordMetrics) add(kind string, instrument metrics.Instrument, value float64, attrs map[string]any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	copied := make(map[string]any, len(attrs))
	maps.Copy(copied, attrs)
	r.calls = append(r.calls, metricCall{kind: kind, instrument: instrument, value: value, attrs: copied})
}

func (r *recordMetrics) Add(_ context.Context, instrument metrics.Instrument, value int64, attrs map[string]any) error {
	r.add("counter", instrument, float64(value), attrs)
	return nil
}

func (r *recordMetrics) Record(_ context.Context, instrument metrics.Instrument, value float64, attrs map[string]any) error {
	r.add("histogram", instrument, value, attrs)
	return nil
}

func (r *recordMetrics) AddUpDown(_ context.Context, instrument metrics.Instrument, value int64, attrs map[string]any) error {
	r.add("updown", instrument, float64(value), attrs)
	return nil
}

func (r *recordMetrics) RecordGauge(_ context.Context, instrument metrics.Instrument, value float64, attrs map[string]any) error {
	r.add("gauge", instrument, value, attrs)
	return nil
}

func (r *recordMetrics) Provider() metric.MeterProvider { return nil }
func (r *recordMetrics) Stop(context.Context) error     { return nil }

func (r *recordMetrics) find(name string) []metricCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []metricCall
	for _, c := range r.calls {
		if c.instrument.Name == name {
			out = append(out, c)
		}
	}
	return out
}

type logCall struct {
	level string
	msg   string
	args  []any
}

type recordLogger struct {
	mu    sync.Mutex
	calls []logCall
}

func (l *recordLogger) add(level, msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls = append(l.calls, logCall{level: level, msg: msg, args: append([]any(nil), args...)})
}

func (l *recordLogger) Debug(_ context.Context, msg string, args ...any) {
	l.add("debug", msg, args...)
}
func (l *recordLogger) Info(_ context.Context, msg string, args ...any) { l.add("info", msg, args...) }
func (l *recordLogger) Warn(_ context.Context, msg string, args ...any) { l.add("warn", msg, args...) }
func (l *recordLogger) Error(_ context.Context, msg string, args ...any) {
	l.add("error", msg, args...)
}

func (l *recordLogger) find(level, msg string) []logCall {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []logCall
	for _, c := range l.calls {
		if c.level == level && c.msg == msg {
			out = append(out, c)
		}
	}
	return out
}

func newTelemetry(t *testing.T) (*recordMetrics, *recordLogger, *tracetest.SpanRecorder, []Option) {
	t.Helper()
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	mc := &recordMetrics{}
	lg := &recordLogger{}
	return mc, lg, recorder, []Option{
		WithMetrics(mc),
		WithTracer(tracer.Wrap(provider)),
		WithLogger(lg),
	}
}

func TestTelemetryLoad(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	mc := &recordMetrics{}
	lg := &recordLogger{}

	p := &fakeProvider{values: validValues(), rev: 42}
	loader, err := New[testConfig](context.Background(), p,
		WithMetrics(mc), WithTracer(tracer.Wrap(provider)), WithLogger(lg))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { _ = loader.Close() })

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("ended spans = %d, want 1", len(spans))
	}
	span := spans[0]
	if span.Name() != "conf.load" {
		t.Errorf("span name = %q, want conf.load", span.Name())
	}
	if span.SpanKind() != trace.SpanKindInternal {
		t.Errorf("span kind = %v, want internal", span.SpanKind())
	}
	attrs := spanAttrs(span.Attributes())
	if attrs["conf.source"] != "fake" {
		t.Errorf("conf.source = %v, want fake", attrs["conf.source"])
	}
	if attrs["conf.revision"] != int64(42) {
		t.Errorf("conf.revision = %v, want 42", attrs["conf.revision"])
	}

	durations := mc.find("conf.reload.duration")
	if len(durations) != 1 {
		t.Fatalf("conf.reload.duration calls = %d, want 1", len(durations))
	}
	if durations[0].kind != "histogram" || durations[0].instrument.Unit != "s" {
		t.Errorf("duration = %s %s, want histogram s", durations[0].kind, durations[0].instrument.Unit)
	}
	if durations[0].attrs["conf.source"] != "fake" {
		t.Errorf("duration conf.source = %v, want fake", durations[0].attrs["conf.source"])
	}
	if _, ok := durations[0].attrs["error.type"]; ok {
		t.Errorf("duration error.type = %v, want absent", durations[0].attrs["error.type"])
	}

	counts := mc.find("conf.reload.count")
	if len(counts) != 1 {
		t.Fatalf("conf.reload.count calls = %d, want 1", len(counts))
	}
	if counts[0].kind != "counter" || counts[0].instrument.Unit != "{reload}" {
		t.Errorf("count = %s %s, want counter {reload}", counts[0].kind, counts[0].instrument.Unit)
	}

	gauges := mc.find("conf.last_success.timestamp")
	if len(gauges) != 1 {
		t.Fatalf("conf.last_success.timestamp calls = %d, want 1", len(gauges))
	}
	if gauges[0].kind != "gauge" || gauges[0].instrument.Unit != "s" {
		t.Errorf("gauge = %s %s, want gauge s", gauges[0].kind, gauges[0].instrument.Unit)
	}
	if gauges[0].value <= 0 {
		t.Errorf("last_success timestamp = %v, want > 0", gauges[0].value)
	}

	if got := lg.find("info", "conf: loaded"); len(got) != 1 {
		t.Errorf("loaded log calls = %d, want 1", len(got))
	}
}

func TestTelemetryParseFailure(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	mc := &recordMetrics{}
	lg := &recordLogger{}

	ch := make(chan Event, 1)
	p := &fakeWatcher{fakeProvider: &fakeProvider{values: validValues(), rev: 42}, ch: ch}
	loader, err := New[testConfig](context.Background(), p,
		WithMetrics(mc), WithTracer(tracer.Wrap(provider)), WithLogger(lg))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { _ = loader.Close() })

	ch <- Event{Key: "checkouts/ttl", Value: "not-a-duration", Revision: 43}
	waitFor(t, func() bool { return len(mc.find("conf.reload.duration")) == 2 })
	if got := loader.State(); got != StateDegraded {
		t.Errorf("State() = %q, want %q", got, StateDegraded)
	}

	spans := recorder.Ended()
	var reloadSpan sdktrace.ReadOnlySpan
	for _, s := range spans {
		if s.Name() == "conf.reload" {
			reloadSpan = s
		}
	}
	if reloadSpan == nil {
		t.Fatal("conf.reload span not recorded")
	}
	if reloadSpan.Status().Code != codes.Error {
		t.Errorf("reload span status = %v, want Error", reloadSpan.Status().Code)
	}
	if got := spanAttrs(reloadSpan.Attributes())["error.type"]; got != "parse" {
		t.Errorf("reload span error.type = %v, want parse", got)
	}

	durations := mc.find("conf.reload.duration")
	if len(durations) != 2 {
		t.Fatalf("conf.reload.duration calls = %d, want 2", len(durations))
	}
	if got := durations[1].attrs["error.type"]; got != "parse" {
		t.Errorf("failure duration error.type = %v, want parse", got)
	}
	if got := len(mc.find("conf.last_success.timestamp")); got != 1 {
		t.Errorf("last_success calls = %d, want 1 (boot only)", got)
	}

	failures := lg.find("warn", "conf: reload failed")
	if len(failures) != 1 {
		t.Fatalf("reload failure logs = %d, want 1", len(failures))
	}
	if !hasArg(failures[0].args, "key", "checkouts/ttl") {
		t.Errorf("reload failure args = %v, want key=checkouts/ttl", failures[0].args)
	}
}

func TestTelemetryWatchError(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	mc := &recordMetrics{}
	lg := &recordLogger{}

	ch := make(chan Event, 1)
	p := &fakeWatcher{fakeProvider: &fakeProvider{values: validValues()}, ch: ch}
	loader, err := New[testConfig](context.Background(), p,
		WithMetrics(mc), WithTracer(tracer.Wrap(provider)), WithLogger(lg))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { _ = loader.Close() })

	ch <- Event{Err: errors.New("watch down"), Revision: 7}
	waitFor(t, func() bool { return len(mc.find("conf.reload.duration")) == 2 })

	var reloadSpan sdktrace.ReadOnlySpan
	for _, s := range recorder.Ended() {
		if s.Name() == "conf.reload" {
			reloadSpan = s
		}
	}
	if reloadSpan == nil {
		t.Fatal("conf.reload span not recorded")
	}
	if got := spanAttrs(reloadSpan.Attributes())["error.type"]; got != "watch" {
		t.Errorf("reload span error.type = %v, want watch", got)
	}

	durations := mc.find("conf.reload.duration")
	if len(durations) != 2 {
		t.Fatalf("conf.reload.duration calls = %d, want 2", len(durations))
	}
	if got := durations[1].attrs["error.type"]; got != "watch" {
		t.Errorf("failure duration error.type = %v, want watch", got)
	}
	if got := lg.find("warn", "conf: watch disconnected"); len(got) != 1 {
		t.Errorf("watch disconnected logs = %d, want 1", len(got))
	}
}

func TestTelemetryHeartbeat(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	mc := &recordMetrics{}
	lg := &recordLogger{}

	ch := make(chan Event, 1)
	p := &fakeWatcher{fakeProvider: &fakeProvider{values: validValues()}, ch: ch}
	loader, err := New[testConfig](context.Background(), p,
		WithMetrics(mc), WithTracer(tracer.Wrap(provider)), WithLogger(lg))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { _ = loader.Close() })

	ch <- Event{Err: errors.New("watch down")}
	waitFor(t, func() bool { return loader.State() == StateDegraded })
	ch <- Event{Heartbeat: true}
	waitFor(t, func() bool { return len(mc.find("conf.last_success.timestamp")) == 2 })
	if got := loader.State(); got != StateLive {
		t.Errorf("State() = %q, want %q", got, StateLive)
	}

	if got := len(mc.find("conf.last_success.timestamp")); got != 2 {
		t.Errorf("last_success calls = %d, want 2 (boot + heartbeat)", got)
	}
	if got := len(mc.find("conf.reload.count")); got != 2 {
		t.Errorf("reload count calls = %d, want 2 (boot + watch error)", got)
	}
	if got := lg.find("info", "conf: watch recovered"); len(got) != 1 {
		t.Errorf("watch recovered logs = %d, want 1", len(got))
	}
}

func TestTelemetryCloseError(t *testing.T) {
	_, lg, _, opts := newTelemetry(t)
	p := &fakeProvider{values: validValues(), closeErr: errors.New("close failed")}
	loader, err := New[testConfig](context.Background(), p, opts...)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := loader.Close(); err == nil {
		t.Fatal("Close() error = nil, want close failure")
	}
	if got := lg.find("error", "conf: close failed"); len(got) != 1 {
		t.Errorf("close failure logs = %d, want 1", len(got))
	}
}

func spanAttrs(attrs []attribute.KeyValue) map[string]any {
	out := make(map[string]any, len(attrs))
	for _, a := range attrs {
		out[string(a.Key)] = a.Value.AsInterface()
	}
	return out
}

func hasArg(args []any, key, value string) bool {
	for i := 0; i+1 < len(args); i += 2 {
		if k, ok := args[i].(string); ok && k == key {
			if v, ok := args[i+1].(string); ok && v == value {
				return true
			}
		}
	}
	return false
}
