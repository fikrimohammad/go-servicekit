// Package errs provides structured errors for Go services.
//
// # When to use
//
// Use errs when a service needs errors that carry a stable integer code, a
// cause, metadata, debug detail, and a stack, and that work with [errors.Is]
// and slog. It is not an error-handling framework; callers still decide HTTP
// status codes and what to log.
//
// # Import
//
//	import "github.com/fikrimohammad/go-servicekit/errs"
//
// # Minimal example
//
//	err := errs.Wrap(503, "fetch profile", cause)
//	if errors.Is(err, errs.New(503, "")) {
//		// handle the code
//	}
//	logger.Error("request failed", errs.SlogAttr(err))
//
// # Lifecycle
//
// An *Error is created by [New], [Wrap], or [AsError]. [Error.WithMeta] and
// [Error.WithDebug] return a modified copy, so the original is safe to share.
// AsError returns the *Error found in a chain, or wraps a plain error with code
// 0.
//
// # Common mistakes
//
//   - Matching by message instead of by non-zero code; errors with code 0
//     never match.
//   - Expecting WithMeta or WithDebug to mutate the receiver.
//   - Putting secrets in debug detail; it is meant for local diagnosis.
//
// # Test recipe
//
// Use errors.Is for codes and errors.As or AsError for fields; the package's
// examples run with go test and show the slog output.
package errs
