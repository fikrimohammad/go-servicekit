// Package metrics installs the metrics pipeline and records counters,
// histograms, up-down counters, and gauges through OTel instruments.
//
// # When to use
//
// Use metrics when a service emits measurements through one client that owns
// the OTLP/gRPC exporter. SDK middleware records through the package-level
// functions, which resolve the current client on every call.
//
// # Import
//
//	import "github.com/fikrimohammad/go-servicekit/observability/metrics"
//
// # Minimal example
//
//	mc, err := metrics.New(ctx, identity.Default(), metrics.Config{})
//	if err != nil {
//		return err
//	}
//	defer mc.Stop(ctx)
//	otel.SetMeterProvider(mc.Provider())
//
//	err = metrics.Add(ctx, metrics.Instrument{
//		Name: "jobs.processed", Unit: "{job}",
//	}, 1, map[string]any{"queue": "default"})
//
// # Lifecycle
//
// [New] validates the config and starts the exporter; the caller installs
// mc.Provider() into the OTel global and calls Stop at shutdown. Stop flushes
// and is idempotent. Config defaults: endpoint localhost:4317, 10-second timeout
// and export interval, and a 160-bucket histogram cap. Instruments are created
// on first use and must keep the same kind, unit, and description.
//
// # Common mistakes
//
//   - Recording before installing the provider, so measurements are dropped.
//   - Reusing a name with a different unit or kind.
//   - Forgetting Stop, so the last batch is lost.
//
// # Test recipe
//
// Use metrics.Wrap(nil) for a no-op client, or wrap a manual reader; the
// package's examples assert no error without a collector.
package metrics
