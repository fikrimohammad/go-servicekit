// Package server builds the HTTP server for SDK services. Every routed request,
// and every request that matches no route, goes through a fixed middleware
// stack that enforces the telemetry contract and problem+json errors.
//
// # When to use
//
// Use server for inbound HTTP at the edge: gateway APIs and vendor webhooks.
// Internal traffic is RPC, so this server distrusts the caller: it starts a new
// root span, extracts only trace context, and validates the request ID.
//
// # Import
//
//	import "github.com/fikrimohammad/go-servicekit/rest/server"
//
// # Minimal example
//
//	srv, err := server.New(server.Config{Addr: ":8080"})
//	if err != nil {
//		return err
//	}
//	srv.Hertz().GET("/ping", func(_ context.Context, c *app.RequestContext) {
//		c.String(http.StatusOK, "pong")
//	})
//	return srv.Run(ctx)
//
// # Lifecycle
//
// [New] validates the config and applies defaults (address :3000, 30-second
// read and write timeouts, 10-second shutdown). [Server.Run] blocks until ctx is
// cancelled, then drains within the shutdown timeout. The stack order is request
// ID, tracing, metrics, logging, recovery, team middleware, handler. An incoming
// X-Request-ID is kept only when it matches the safe pattern; otherwise the
// server generates one.
//
// # Common mistakes
//
//   - Adding team middleware after routes; Hertz applies Use only to later
//     routes.
//   - Expecting a 4xx to set the span status to Error; only 5xx does.
//   - Hand-writing error JSON instead of [Error], which keeps the RFC 9457
//     shape.
//
// # Test recipe
//
// Start Run on a free port and call it with net/http; assert the problem body,
// the request ID header, and the middleware order through a real request.
package server
