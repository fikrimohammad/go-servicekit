// Package engine schedules and runs cron jobs from the cron package. It wraps
// robfig/cron and runs each job through a middleware chain that applies the
// toggler, distributed locker, timeout, panic recovery, logging, metrics, and
// tracing.
//
// # When to use
//
// Use engine when a service needs in-process recurring jobs with telemetry and
// optional distributed locking. It is not a workflow engine; jobs are
// independent and fire on a schedule.
//
// # Import
//
//	import "github.com/fikrimohammad/go-servicekit/cron/engine"
//
// # Minimal example
//
//	eng, err := engine.New(engine.Config{})
//	if err != nil {
//		return err
//	}
//	if err := eng.Register(job); err != nil {
//		return err
//	}
//	eng.Start()
//	defer eng.Stop(ctx)
//
// # Lifecycle
//
// [New] validates the config and applies defaults: a one-minute global timeout
// and a ten-second shutdown timeout. [Engine.Start] begins scheduling;
// [Engine.Stop] waits for in-flight runs and is safe to call twice.
// [Engine.Trigger] runs a registered job synchronously. [WithToggler],
// [WithLocker], [WithMetrics], and [WithTracer] inject collaborators; metrics
// and tracing default to the OTel globals.
//
// # Common mistakes
//
//   - Registering after Start; registration closes once the engine runs.
//   - Reusing a job name; Register rejects duplicates.
//   - Expecting Trigger to run asynchronously; it waits for the job.
//
// # Test recipe
//
// Use Trigger for deterministic runs with a fake toggler or locker; no timers
// or sleeps are needed, as the package's examples show.
package engine
