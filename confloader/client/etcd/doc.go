// Package etcd implements a confloader client backed by go.etcd.io/etcd/client/v3.
//
// # When to use
//
// Use etcd when configuration lives in etcd and must be read through
// confloader. Most services select it with confloader.Config{Provider:
// confloader.ProviderEtcd} instead of importing this package. The storage model
// is namespace, then folder, then key; a key never lives directly under the
// root, and folders do not nest.
//
// # Import
//
//	import "github.com/fikrimohammad/go-servicekit/confloader/client/etcd"
//
// # Minimal example
//
//	c, err := etcd.New(etcd.Config{
//		Endpoint:         "localhost:2379",
//		AuthClientID:     "username",
//		AuthClientSecret: "password",
//		Namespace:        "my-project",
//	})
//	if err != nil {
//		return err
//	}
//	defer c.Close()
//
// # Lifecycle
//
// [New] opens one etcd client with a ten-second dial timeout and static
// username/password credentials. [Client.Fetch] reads a single key and derives
// its revision from etcd's mod revision. Close releases the connection. Each
// Fetch reads one key, the loader falls back to the default folder, and Fetch
// is safe for concurrent use.
//
// # Common mistakes
//
//   - Calling New per request instead of once at startup.
//   - Leaving Namespace empty; confloader rejects that config.
//   - Building a key path with extra segments; the adapter expects exactly
//     namespace/folder/key.
//
// # Test recipe
//
// Use confloader's fake client for unit tests; this adapter is thin and needs a
// live etcd instance.
package etcd
