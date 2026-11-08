// Package tracer installs the trace pipeline and exposes named tracers,
// attribute conversion, trace and span ID extraction, and global setup.
//
// # When to use
//
// Use tracer when a service starts spans through one client that owns the
// OTLP/gRPC exporter and the W3C propagator. Middleware and the composition root
// take a tracer.Client, so tests can inject a no-op.
//
// # Import
//
//	import "github.com/fikrimohammad/go-servicekit/observability/tracer"
//
// # Minimal example
//
//	tc, err := tracer.New(ctx, identity.Default(), tracer.Config{})
//	if err != nil {
//		return err
//	}
//	defer tc.Stop(ctx)
//	tracer.SetGlobal(tc)
//
//	ctx, span := tc.Tracer("orders").Start(ctx, "create")
//	defer span.End()
//	traceID := tracer.TraceIDFrom(ctx)
//
// # Lifecycle
//
// [New] validates the config and starts the batch processor; [SetGlobal]
// installs the provider and the W3C TraceContext and Baggage propagator. Stop
// flushes and is idempotent. Config defaults: endpoint localhost:4317, 5-second
// export and batch timeouts, queue size 2048, export batch 512; a nil Sampler
// selects the parent-based always-on default.
//
// # Common mistakes
//
//   - Starting spans before SetGlobal, so they go to the no-op provider.
//   - Calling SetGlobal twice and leaking the previous provider.
//   - Using TraceIDFrom outside a valid span; it returns an empty ID.
//
// # Test recipe
//
// Use tracer.Wrap(nil) for a no-op client, or a manual span recorder; the
// package's example asserts IDs without a collector.
package tracer
