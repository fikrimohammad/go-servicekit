// Package redis wraps github.com/redis/go-redis/v9 and records OpenTelemetry
// traces and metrics for every executed command.
//
// # When to use
//
// Use redis when a service needs Redis with consistent telemetry and a managed
// connection. It returns the full go-redis command set, so it is not a cache
// abstraction; callers choose keys, TTLs, and data shapes.
//
// # Import
//
//	import "github.com/fikrimohammad/go-servicekit/redis"
//
// # Minimal example
//
//	client, err := redis.New(ctx, redis.Config{Addrs: []string{"localhost:6379"}})
//	if err != nil {
//		return err
//	}
//	defer client.Close()
//
//	if err := client.Set(ctx, "session:42", "active", 10*time.Minute).Err(); err != nil {
//		return err
//	}
//	val, err := client.Get(ctx, "session:42").Result()
//	if errors.Is(err, goredis.Nil) {
//		// key is missing, not a failure
//	}
//
// # Lifecycle
//
// [New] builds the client, attaches the instrumentation hook, and pings the
// server. The returned go-redis client is safe for concurrent use, and its
// Close stops the pool-metrics poller and waits for it. Config covers
// standalone, sentinel, and cluster modes; [WithMetrics] and [WithTracer]
// override the global OTel providers.
//
// # Common mistakes
//
//   - Forgetting Close, or treating goredis.Nil as an error.
//   - Assuming telemetry carries query text; values are redacted as "?".
//   - Mixing cluster mode with a non-zero DB.
//
// # Test recipe
//
// Start miniredis and call New with its address; the package's tests exercise
// the real New path, so no live Redis server is needed.
package redis
