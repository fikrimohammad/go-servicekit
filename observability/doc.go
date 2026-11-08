// Package observability unifies logging, metrics, and tracing for Go services.
// Each signal is a separate package; this root has no API and is the wiring
// guide.
//
// # When to use
//
// Use this guide when wiring telemetry once at startup. Applications use the
// logs, metrics, tracer, and attributes subpackages directly; importing this
// root gives an empty package.
//
// # Import
//
// The root has no importable API. Import the subpackages:
//
//	github.com/fikrimohammad/go-servicekit/observability/logs
//	github.com/fikrimohammad/go-servicekit/observability/metrics
//	github.com/fikrimohammad/go-servicekit/observability/tracer
//	github.com/fikrimohammad/go-servicekit/observability/attributes
//
// # Minimal example
//
//	info := identity.Default()
//	logger, _ := logs.New(info, logs.Config{Format: logs.FormatJSON})
//	logs.SetDefault(logger)
//	mc, _ := metrics.New(ctx, info, metrics.Config{Endpoint: "collector:4317"})
//	otel.SetMeterProvider(mc.Provider())
//	tc, _ := tracer.New(ctx, info, tracer.Config{Endpoint: "collector:4317"})
//	tracer.SetGlobal(tc)
//	defer mc.Stop(ctx)
//	defer tc.Stop(ctx)
//
// # Lifecycle
//
// The caller owns each client and calls Stop at shutdown; Stop flushes and is
// idempotent. Logs use a package default; metrics and traces use the OTel
// globals. Package-level functions resolve the current global on every call, so
// an install or swap takes effect immediately. Before any install, metrics and
// traces are no-ops.
//
// # Common mistakes
//
//   - Importing this root and expecting functions.
//   - Installing the providers after emitting the first signals.
//   - Forgetting Stop, so pending batches are dropped.
//
// # Test recipe
//
// Use a discard writer for logs and skip exporter assertions; the subpackage
// tests show the wiring without a collector.
package observability
