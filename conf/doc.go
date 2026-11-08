// Package conf loads and watches typed configuration from pluggable
// providers.
//
// # When to use
//
// Use conf when a service has two kinds of configuration: a boot set loaded
// once from a file, and a hot set that must change without a restart. One
// loader binds one provider, every leaf is a typed Getter, reads never touch
// the network, boot is strict, and State reports the truth.
//
// # Import
//
//	import (
//		"github.com/fikrimohammad/go-servicekit/conf"
//		"github.com/fikrimohammad/go-servicekit/conf/providers/etcd"
//		"github.com/fikrimohammad/go-servicekit/conf/providers/file"
//	)
//
// # Minimal example
//
//	type Boot struct {
//		Checkout struct {
//			TTL conf.Getter[time.Duration] `conf:"name=ttl"`
//		} `conf:"folder=checkouts"`
//	}
//
//	loader, err := conf.New[Boot](ctx, file.New(file.Config{
//		Path:   "disburseapp.prod.yaml",
//		Format: conf.FormatYAML,
//	}))
//	if err != nil {
//		return err
//	}
//	defer loader.Close()
//
//	ttl := loader.Data().Checkout.TTL.GetWithDefault(30 * time.Second)
//
// # Key derivation
//
// A config type is a tree. Nested plain structs are folders; Getter leaves are
// keys. Keys are the snake_case field path:
//
//	type Hot struct {
//		Checkout struct {
//			MaxRetries conf.Getter[int] `conf:"name=max_retry,format=scalar"`
//		} `conf:"folder=checkouts"`
//	}
//
// produces checkouts/max_retry. A conf name tag renames a leaf segment, a conf
// folder tag renames a folder, and conf:"-" skips a field or subtree. conf:"-"
// on a Getter is an error because every Getter must be bound.
//
// # Formats
//
// A file provider owns its document format through its Config and implements
// Formatter.Parse; leaves must not carry a format tag. An etcd provider owns no
// format; every leaf must declare a format tag: yaml, json, toml, or scalar.
// There is no format sniffing.
//
// # Lifecycle
//
// New loads the provider once, requires every key, decodes every value, and
// binds the Getters. An etcd provider then watches from the load revision. A
// bad update keeps last-known-good and sets StateDegraded; a later good value
// returns StateLive. Reads are per-key atomic; two reads can span a reload.
// Close cancels the watch, waits for it, and closes the provider. Reads after
// close return ErrClosed, and State is StateClosed.
//
// # Common mistakes
//
//   - Putting conf:"-" on a Getter.
//   - Forgetting to create an etcd key before deploying its code.
//   - Omitting the format tag on an etcd leaf.
//   - Expecting two reads to see the same revision.
//   - Logging values; the library logs keys and revisions only.
//
// # Test recipe
//
// Use a temp file for providers/file; a fake Provider and Watcher for loader
// tests; CONF_ETCD_ENDPOINT for the integration tests. The package examples run
// with go test.
package conf
