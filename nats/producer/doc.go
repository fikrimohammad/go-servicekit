// Package producer publishes messages to NATS JetStream streams. A [Producer]
// owns the stream of each registered topic and sends messages synchronously,
// delayed, or asynchronously.
//
// # When to use
//
// Use producer when a service emits domain events to JetStream and needs
// at-least-once publishing with telemetry. It is not an RPC client; events are
// addressed by topic and tag, not by a single consumer.
//
// # Import
//
//	import "github.com/fikrimohammad/go-servicekit/nats/producer"
//
// # Minimal example
//
//	p, err := producer.New(identity.Info{Name: "orders-service"})
//	if err != nil {
//		return err
//	}
//	if err := p.Register(producer.Config{
//		Endpoints: []string{"nats://localhost:4222"},
//		Topic:     "orders",
//	}); err != nil {
//		return err
//	}
//	if err := p.Start(ctx); err != nil {
//		return err
//	}
//	defer p.Shutdown(ctx)
//
//	return p.PublishSync(ctx, producer.Message{
//		Topic: "orders", Tag: "created", Key: "order-42", Body: body,
//	})
//
// # Lifecycle
//
// [New] validates the service name; [Producer.Register] stores one topic config
// and refuses to run after Start. [Producer.Start] dials and ensures the stream;
// [Producer.Shutdown] drains pending publishes. [Producer.PublishAsync] reports
// through a callback, while PublishSync and PublishSyncWithDelay wait. Config
// defaults to a ten-second send timeout.
//
// # Common mistakes
//
//   - Publishing to a topic that was never registered.
//   - Calling Register after Start; it returns ErrAlreadyStarted.
//   - Treating PublishAsync as fire-and-forget; its callback still runs.
//
// # Test recipe
//
// Start the embedded NATS server with JetStream, Register, Start, then publish
// and consume through a JetStream context; no external broker is needed.
package producer
