// Package attributes provides helpers for normalizing developer-style
// attribute keys and converting values to OpenTelemetry attributes.
//
// # When to use
//
// Use attributes when a service emits keys like order_id and wants the OTel
// style order.id, or when converting map[string]any values into attribute
// key-values. The logs, metrics, and tracer packages call it internally. The
// helpers are stateless, so they are safe on any hot path.
//
// # Import
//
//	import "github.com/fikrimohammad/go-servicekit/observability/attributes"
//
// # Minimal example
//
//	key := attributes.NormalizeKey("order_id") // "order.id"
//	kvs := attributes.ConvertMapsToKVs(map[string]any{"order_id": 42})
//
// # Lifecycle
//
// The helpers are pure functions with no state. NormalizeKey preserves keys
// that already contain a dot and the correlation keys trace_id, span_id, and
// trace_flags. ConvertMapsToKVs merges its maps in argument order, later values
// winning, normalizes each key, and sorts the result by key.
//
// # Common mistakes
//
//   - Hand-normalizing keys instead of calling NormalizeKey.
//   - Expecting a dotted key to be rewritten; dotted keys are already OTel
//     style.
//   - Expecting insertion order; ConvertMapsToKVs sorts by key.
//
// # Test recipe
//
// Table-test NormalizeKey with the preserved cases and ConvertMapsToKVs with
// mixed value types; the package's examples run with go test.
package attributes
