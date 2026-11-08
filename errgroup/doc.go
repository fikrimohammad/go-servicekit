// Package errgroup provides a concurrency-limited, panic-safe wrapper around
// golang.org/x/sync/errgroup.
//
// # When to use
//
// Use errgroup when a service runs a bounded fan-out, wants the first error to
// cancel the rest, and needs a panic in one task to surface as an error instead
// of crashing the process. The group follows the upstream first-error and
// cancellation rules and is safe for concurrent use.
//
// # Import
//
//	import "github.com/fikrimohammad/go-servicekit/errgroup"
//
// # Minimal example
//
//	g := errgroup.New(ctx, errgroup.WithMaxConcurrency(4))
//	for _, item := range items {
//		g.Go(func(ctx context.Context) error {
//			return process(ctx, item)
//		})
//	}
//	if err := g.Wait(); err != nil {
//		return err
//	}
//
// # Lifecycle
//
// [New] derives the group context; [Group.Wait] joins every task and returns
// the first error or panic. [Group.Go] always schedules; [Group.TryGo] returns
// false when the limit is full. [Group.SubGroup] inherits the parent's context
// and concurrency limit, and its error does not cancel the parent.
//
// # Common mistakes
//
//   - Using the zero Group; always construct with New.
//   - Capturing a loop variable instead of a parameter in the task.
//   - Expecting a child error to cancel the parent.
//
// # Test recipe
//
// Use go test; assert the first error, check [PanicError] with errors.As, and
// use TryGo to test saturation without timing.
package errgroup
