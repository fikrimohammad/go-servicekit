// Package logs installs the application logger, routes structured records to
// stdout/stderr by severity, and adds trace correlation fields from a record's
// context.
//
// # When to use
//
// Use logs for every application record. The package keeps a default logger, so
// the package-level functions work without wiring, and [Logger] methods add
// trace_id and span_id from a valid span context.
//
// # Import
//
//	import "github.com/fikrimohammad/go-servicekit/observability/logs"
//
// # Minimal example
//
//	logger, err := logs.New(identity.Default(), logs.Config{Format: logs.FormatJSON})
//	if err != nil {
//		return err
//	}
//	logs.SetDefault(logger)
//	logs.Info(ctx, "order accepted", "order_id", "o-1")
//
// # Lifecycle
//
// [New] validates the config and returns a [Logger]; SetDefault installs it and
// Default returns the current one. Before SetDefault, Default is a debug-level
// text logger on stderr and is never nil. Records with a valid span context get
// trace_id and span_id; an explicit key wins.
//
// # Common mistakes
//
//   - Assuming the pre-install default matches your format; it is text on
//     stderr.
//   - Logging startup records before installing the production logger, so they
//     use the default format.
//   - Building a writer per request instead of reusing the logger.
//
// # Test recipe
//
// Pass io.Discard writers in Config and assert the record through a custom
// handler; the package's example runs with go test.
package logs
