// Package client builds the HTTP client that SDK services use to call external
// vendors. It enforces the retry, deadline, and telemetry rules of the rest
// module, so every outbound call behaves the same.
//
// # When to use
//
// Use client for every outbound call to a vendor from a gateway service. It is
// the external boundary, so it distrusts the other side: no trace context or
// request ID is sent, and query strings are redacted in telemetry.
//
// # Import
//
//	import "github.com/fikrimohammad/go-servicekit/rest/client"
//
// # Minimal example
//
//	c, err := client.New(client.Config{AttemptTimeout: 5 * time.Second})
//	if err != nil {
//		return err
//	}
//
//	req := protocol.AcquireRequest()
//	defer protocol.ReleaseRequest(req)
//	resp := protocol.AcquireResponse()
//	defer protocol.ReleaseResponse(resp)
//	req.SetMethod(http.MethodGet)
//	req.SetRequestURI("https://vendor.example.com/v1/items")
//
//	err = c.Do(ctx, req, resp)
//
// # Lifecycle
//
// [New] returns the plain Hertz client with the chain installed: logging,
// retry, tracing, metrics, deadline, then team middleware. Build it once and
// share it. Config defaults: 30-second overall timeout, 10-second attempt
// timeout, three attempts, full-jitter backoff, Retry-After capped at ten
// seconds. A ctx deadline takes precedence, and timeouts satisfy
// errors.Is(err, context.DeadlineExceeded).
//
// # Common mistakes
//
//   - Adding retry again with SetRetryIfFunc or on top of a RoundTripper.
//   - Retrying a non-idempotent POST without an Idempotency-Key.
//   - Expecting a nil error for a 4xx or a final 5xx; check the status.
//
// # Test recipe
//
// Use httptest for the vendor and assert one span per attempt with
// http.request.resend_count, Retry-After handling, and an early return with
// errors.Is(err, context.DeadlineExceeded).
package client
