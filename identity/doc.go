// Package identity provides the standardized service identity attached to logs,
// metrics, and traces.
//
// # When to use
//
// Use identity to resolve service.name, service.version, and
// deployment.environment.name once at startup and pass them to observability.
// It reads APP_NAME, APP_VERSION, and APP_ENV, with the OpenTelemetry resource
// conventions as fallbacks, so the same binary works in local, staging, and
// production.
//
// # Import
//
//	import "github.com/fikrimohammad/go-servicekit/identity"
//
// # Minimal example
//
//	info := identity.Resolve(nil)
//	if err := info.Validate(); err != nil {
//		return err
//	}
//	attrs := info.Attributes()
//
// # Lifecycle
//
// Resolve and Default are pure lookups; Placeholder returns the development
// fallback. Info is a small value type, safe to copy into configs and log
// fields. [Info.Validate] rejects an identity with no service name.
// [Info.Attributes] returns the semantic-convention keys, with the legacy
// environment key normalized.
//
// # Common mistakes
//
//   - Skipping Validate and exporting an empty service.name.
//   - Reading the environment in many places instead of resolving once.
//   - Expecting Resolve to fail; it returns a placeholder rather than an error.
//
// # Test recipe
//
// Resolve accepts a lookup function, so tests pass a map instead of mutating
// the environment; the package's examples run with go test.
package identity
