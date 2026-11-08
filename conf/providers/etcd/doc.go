// Package etcd reads and watches one etcd namespace.
//
// # When to use
//
// Use etcd for hot configuration. The provider owns one keyspace; every leaf in
// the config struct declares its own format because etcd values are raw
// strings.
//
// # Import
//
//	import "github.com/fikrimohammad/go-servicekit/conf/providers/etcd"
//
// # Minimal example
//
//	p := etcd.New(etcd.Config{
//		Endpoint:  os.Getenv("ETCD_ENDPOINT"),
//		Namespace: "disburseapp",
//	})
//	loader, err := conf.New[Hot](ctx, p)
//	if err != nil {
//		return err
//	}
//	defer loader.Close()
//
// # Lifecycle
//
// [Provider.Load] opens one client, reads the namespace prefix once, and
// returns the requested keys plus the header revision. [Provider.Watch] streams
// changes from that revision, with progress-notify heartbeats, reconnect and
// backoff, compaction detection, and a full resync. [Provider.Close] waits for
// the watch and closes the client; it is idempotent.
//
// # Common mistakes
//
//   - Omitting format= on a leaf; the etcd provider does not own a format.
//   - Putting a format in the provider config; it belongs on each leaf.
//   - Deploying the code before creating the key; boot is strict.
//
// # Test recipe
//
// Set CONF_ETCD_ENDPOINT and run the integration tests; without it they skip.
// Unit tests cover config validation, key mapping, and the lifecycle guards.
package etcd
