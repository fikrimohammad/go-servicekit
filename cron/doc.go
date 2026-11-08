// Package cron defines the job model shared by the cron engine and its
// plug-ins: a [Job] with a [HandlerFunc], the [Toggler] that can disable a
// job, and the [Locker] and [LockToken] that keep a job on one worker at a
// time.
//
// # When to use
//
// Use cron to declare jobs and the interfaces the engine needs for toggling and
// distributed locking. Scheduling and running live in the engine subpackage;
// this package is data and contracts only.
//
// # Import
//
//	import "github.com/fikrimohammad/go-servicekit/cron"
//
// # Minimal example
//
//	job := cron.Job{
//		Name:     "nightly-report",
//		Schedule: "@every 1h",
//		Handler:  func(ctx context.Context) error { return generate(ctx) },
//	}
//	// Pass the job to engine.Register.
//
// # Lifecycle
//
// A [Job] is a value: fill Name, Schedule, and Handler, plus an optional
// Timeout that overrides the engine's global timeout. A [Toggler] is checked
// before each run; a [Locker] acquires a lock whose TTL covers the job timeout,
// and its fencing token travels in the context.
//
// # Common mistakes
//
//   - Leaving Name, Schedule, or Handler empty; all three are required.
//   - Scheduling jobs from this package directly; it has no runner.
//   - Ignoring the fencing token when the work reaches a downstream system.
//
// # Test recipe
//
// Build a Job and run it through the engine's Trigger synchronously, so tests
// need no timers; the package's examples cover the fencing-token context.
package cron
