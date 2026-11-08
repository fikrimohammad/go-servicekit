// Package confloader loads and watches hot configuration and secrets from
// pluggable backends (etcd, Infisical).
//
// # When to use
//
// Use confloader when a service needs typed config values that refresh without a
// restart. Declare a struct of [Getter] fields with conf tags; the loader
// fetches them eagerly and keeps them fresh by polling. Reads never block on
// that loop: a cold or stale value is fetched directly, so Get is safe on the
// request path.
//
// # Import
//
//	import "github.com/fikrimohammad/go-servicekit/confloader"
//
// # Minimal example
//
//	type AppConfig struct {
//		Debug Getter[bool] `conf:"folder=settings,key=debug"`
//	}
//
//	loader, err := confloader.New[AppConfig](ctx, cfg)
//	if err != nil {
//		return err
//	}
//	defer loader.Stop()
//
//	debug, err := loader.Data().Debug.Get(ctx)
//
// # Lifecycle
//
// [New] validates the provider config, applies the watcher defaults (30-second
// poll, three retries), fetches every entry unless WithInitialFetch(false), and
// starts the polling loop. Stop ends it. [Getter.Get] refreshes a stale entry on
// read; GetWithDefault never fails.
//
// # Common mistakes
//
//   - Omitting the conf tag, or putting it on a non-Getter field.
//   - Expecting Get to return ErrStale; it returns the source's own error.
//   - Forgetting Stop, or reading the environment instead of the loader.
//
// # Test recipe
//
// Implement client.Client with a map-backed fake and pass it with WithClient;
// the package's example runs with go test and no etcd or Infisical.
package confloader
