package eval

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.opentelemetry.io/otel"

	"github.com/fikrimohammad/go-servicekit/identity"
	"github.com/fikrimohammad/go-servicekit/observability/logs"
	"github.com/fikrimohammad/go-servicekit/observability/metrics"
	"github.com/fikrimohammad/go-servicekit/observability/tracer"
)

// wire installs logs, metrics, and tracing and returns a function that stops
// the two clients. The test then starts a span through the package-level tracer
// and requires it to be recording.
//
// TODO: install the logger, the meter provider, and the tracer as the contracts
// describe, and return a stop function.
func wire(_ context.Context, _ identity.Info) (func(context.Context), error) {
	_ = otel.GetMeterProvider()
	_ = logs.Config{}
	_ = metrics.Config{}
	_ = tracer.Config{}
	return nil, errors.New("not implemented")
}

func TestWire(t *testing.T) {
	info := identity.Info{Name: "eval", Version: "v1", Environment: "test"}
	stop, err := wire(context.Background(), info)
	if err != nil {
		t.Fatalf("wire: %v", err)
	}
	if stop == nil {
		t.Fatal("wire returned a nil stop function")
	}
	stopCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	defer stop(stopCtx)

	ctx, span := tracer.Tracer("eval").Start(context.Background(), "operation")
	defer span.End()
	if !span.IsRecording() {
		t.Fatal("span is not recording; the trace provider was not installed")
	}
	if id := tracer.TraceIDFrom(ctx); id == "" {
		t.Error("trace id is empty")
	}
	if err := metrics.Add(ctx, metrics.Instrument{Name: "eval.runs", Unit: "{run}"}, 1, nil); err != nil {
		t.Errorf("metrics.Add: %v", err)
	}
}
