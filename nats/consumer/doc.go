// Package consumer runs durable NATS JetStream consumers. A [Consumer] passes
// each delivered message to a [HandlerFunc] and settles the message from the
// handler's error.
//
// # When to use
//
// Use consumer when a service consumes JetStream messages with at-least-once
// delivery, telemetry, and bounded concurrency. It is not a request/reply
// transport; handlers must be idempotent because redelivery is expected.
//
// # Import
//
//	import "github.com/fikrimohammad/go-servicekit/nats/consumer"
//
// # Minimal example
//
//	c, err := consumer.New(identity.Info{Name: "orders-service"})
//	if err != nil {
//		return err
//	}
//	err = c.Register(consumer.Config{
//		Endpoints: []string{"nats://localhost:4222"},
//		Topic:     "orders",
//		Group:     "billing",
//	}, func(ctx context.Context, msg consumer.Message) error {
//		return process(ctx, msg.Body())
//	})
//	if err != nil {
//		return err
//	}
//	return c.Start(ctx)
//
// # Lifecycle
//
// [New] validates the service name; [Consumer.Register] stores one topic/group
// config and its handler and refuses to run after Start. [Consumer.Start] dials
// and begins pulling; [Consumer.Shutdown] drains in-flight handlers. Config
// defaults: 60-second AckWait, five deliveries, ten concurrent handlers, tags
// ["*"], and consume from last.
//
// # Common mistakes
//
//   - Running a non-idempotent handler; delivery is at least once.
//   - Returning a permanent error without wrapping ErrNoRetry, so a poison
//     message redelivers until the delivery limit.
//   - Registering after Start; it returns ErrAlreadyStarted.
//
// # Test recipe
//
// Start the embedded NATS server the package's tests use, with JetStream and a
// temporary store, then Register and Start; no external broker is needed.
package consumer
