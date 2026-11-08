// Package client defines the shared contract every config backend (etcd,
// Infisical) implements.
//
// # When to use
//
// Use client when implementing a new config backend for confloader, or when
// writing a test fake. Services normally use the confloader package and a
// built-in backend, not this interface directly. Keeping the interface in its
// own package also breaks the import cycle between the loader and the concrete
// backends.
//
// # Import
//
//	import "github.com/fikrimohammad/go-servicekit/confloader/client"
//
// # Minimal example
//
//	type memory struct{ m map[string]string }
//
//	func (c memory) Fetch(ctx context.Context, folder, key string) (client.Fetched, error) {
//		v, ok := c.m[folder+"/"+key]
//		if !ok {
//			return client.Fetched{}, client.ErrNotFound
//		}
//		return client.Fetched{Value: v, Revision: "1"}, nil
//	}
//
//	func (c memory) Close() error { return nil }
//
// # Lifecycle
//
// The loader calls Fetch on a polling loop and Close on Stop. Fetch must return
// ErrNotFound for a genuinely absent key, never an empty Fetched with a nil
// error; the loader falls back to the "default" folder once before treating a
// key as absent.
//
// # Common mistakes
//
//   - Returning a nil error for a missing key.
//   - Omitting Revision, so the loader cannot tell a refresh from a change.
//   - Blocking forever or panicking in Fetch; it runs in the polling loop.
//
// # Test recipe
//
// Implement the two methods on a map; no backend is needed. The confloader
// package's example does exactly that.
package client
