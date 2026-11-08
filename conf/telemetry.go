package conf

import (
	"context"
	"errors"
	"time"

	"github.com/fikrimohammad/go-servicekit/observability/logs"
	"github.com/fikrimohammad/go-servicekit/observability/metrics"
	"github.com/fikrimohammad/go-servicekit/observability/tracer"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var (
	reloadDurationInstrument = metrics.Instrument{
		Name:        "conf.reload.duration",
		Unit:        "s",
		Description: "Duration of configuration load and reload operations.",
	}
	reloadCountInstrument = metrics.Instrument{
		Name:        "conf.reload.count",
		Unit:        "{reload}",
		Description: "Number of configuration load and reload operations.",
	}
	lastSuccessInstrument = metrics.Instrument{
		Name:        "conf.last_success.timestamp",
		Unit:        "s",
		Description: "Unix timestamp of the last successful configuration sync.",
	}
)

func resolveMetrics(m metrics.Client) metrics.Client {
	if m != nil {
		return m
	}
	return metrics.Default()
}

func resolveTracer(t tracer.Client) tracer.Client {
	if t != nil {
		return t
	}
	return tracer.Default()
}

func resolveLogger(l logs.Logger) logs.Logger {
	if l != nil {
		return l
	}
	return logs.Default()
}

func (c *loaderCore) metricsClient() metrics.Client { return resolveMetrics(c.metrics) }
func (c *loaderCore) tracerClient() tracer.Client   { return resolveTracer(c.tracer) }
func (c *loaderCore) loggerClient() logs.Logger     { return resolveLogger(c.logger) }

// errorClass maps an error to a low-cardinality error.type value.
func errorClass(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrInvalidConfig),
		errors.Is(err, ErrInvalidTag),
		errors.Is(err, ErrInvalidType),
		errors.Is(err, ErrDuplicateKey),
		errors.Is(err, ErrInvalidFormat),
		errors.Is(err, ErrMissingFormat):
		return "config"
	case errors.Is(err, ErrMissingKey),
		errors.Is(err, ErrUnknownKey),
		errors.Is(err, ErrParse):
		return "parse"
	case errors.Is(err, ErrConnect):
		return "connect"
	default:
		return "_OTHER"
	}
}

func recordMetric(ctx context.Context, mc metrics.Client, source string, start time.Time, err error, class string) {
	attrs := map[string]any{"conf.source": source}
	if err != nil {
		attrs["error.type"] = class
	}
	_ = mc.Record(ctx, reloadDurationInstrument, time.Since(start).Seconds(), attrs)
	_ = mc.Add(ctx, reloadCountInstrument, 1, attrs)
	if err == nil {
		_ = mc.RecordGauge(ctx, lastSuccessInstrument, float64(time.Now().Unix()), map[string]any{"conf.source": source})
	}
}

func recordReload(ctx context.Context, mc metrics.Client, tc tracer.Client, source string, start time.Time, rev int64, err error, class string) {
	ctx, span := tc.Tracer("conf").Start(ctx, "conf.reload", trace.WithSpanKind(trace.SpanKindInternal))
	span.SetAttributes(attribute.String("conf.source", source))
	if rev != 0 {
		span.SetAttributes(attribute.Int64("conf.revision", rev))
	}
	if err != nil {
		span.SetAttributes(attribute.String("error.type", class))
		span.SetStatus(codes.Error, class)
	}
	span.End()
	recordMetric(ctx, mc, source, start, err, class)
}

func recordHeartbeat(ctx context.Context, mc metrics.Client, source string) {
	_ = mc.RecordGauge(ctx, lastSuccessInstrument, float64(time.Now().Unix()), map[string]any{"conf.source": source})
}

func (c *loaderCore) recordReload(ctx context.Context, rev int64, start time.Time, err error, class string) {
	recordReload(ctx, c.metricsClient(), c.tracerClient(), c.source, start, rev, err, class)
}

func (c *loaderCore) recordHeartbeat(ctx context.Context) {
	recordHeartbeat(ctx, c.metricsClient(), c.source)
}

func startLoadSpan(ctx context.Context, tc tracer.Client, source string) (context.Context, trace.Span) {
	ctx, span := tc.Tracer("conf").Start(ctx, "conf.load", trace.WithSpanKind(trace.SpanKindInternal))
	span.SetAttributes(attribute.String("conf.source", source))
	return ctx, span
}

func endLoadSpan(span trace.Span, err error) {
	if err != nil {
		class := errorClass(err)
		span.SetAttributes(attribute.String("error.type", class))
		span.SetStatus(codes.Error, class)
	}
	span.End()
}
