// Package conf loads and watches typed configuration from file or etcd.
//
// # When to use
//
// Use conf when a service has two kinds of configuration: a boot set loaded
// once from a file, and a hot set that must change without a restart. One
// loader binds one source, every leaf is a typed Getter, reads never touch the
// network, boot is strict, and State reports the truth.
//
// # Import
//
//	import "github.com/fikrimohammad/go-servicekit/conf"
//
// # Minimal example
//
//	type Boot struct {
//		Checkout struct {
//			TTL conf.Getter[time.Duration] `conf:"name=ttl"`
//		} `conf:"folder=checkouts"`
//	}
//
//	loader, err := conf.New[Boot](ctx, conf.File(conf.FileConfig{
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
// Use [Etcd] instead of [File] for a watched etcd source:
//
//	loader, err := conf.New[Hot](ctx, conf.Etcd(conf.EtcdConfig{
//		Endpoint:  os.Getenv("ETCD_ENDPOINT"),
//		Namespace: "disburseapp",
//	}))
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
// A file source owns its document format through FileConfig and applies it to
// the whole document; leaves must not carry a format tag. An etcd source owns
// no format; every leaf must declare a format tag: yaml, json, toml, or scalar.
// There is no format sniffing.
//
// # Lifecycle
//
// New loads the source once, requires every key, decodes every value, and binds
// the Getters. An etcd source then watches from the load revision. The loader
// owns the watch: canceling the context passed to New stops the boot load but
// not the watch; only Close stops it. A bad update keeps last-known-good and
// sets StateDegraded; a later good value returns StateLive. Reads are per-key
// atomic; two reads can span a reload. Getter.Get returns a deep copy, so
// callers can mutate maps, slices, and pointers without changing the snapshot.
// Close cancels the watch, waits for it, and closes the source. Reads after
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
// Use a temp file with [File]; set CONF_ETCD_ENDPOINT and use [Etcd] for the
// integration tests. The loader tests use fake providers. The package examples
// run with go test.
package conf
