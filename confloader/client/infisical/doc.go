// Package infisical implements a confloader client backed by the Infisical Go
// SDK.
//
// # When to use
//
// Use infisical when secrets live in Infisical and must be read through
// confloader. Most services select it with confloader.Config{Provider:
// confloader.ProviderInfisical}; it needs a workspace, an environment, and a
// secret path. The adapter is a thin mapping over the SDK; it adds revision
// tracking so the loader can detect changes without re-parsing secrets.
//
// # Import
//
//	import "github.com/fikrimohammad/go-servicekit/confloader/client/infisical"
//
// # Minimal example
//
//	c, err := infisical.New(infisical.Config{
//		Endpoint:         "https://app.infisical.com",
//		AuthClientID:     "client-id",
//		AuthClientSecret: "client-secret",
//		Namespace:        "workspace-id",
//		Environment:      "prod",
//	})
//	if err != nil {
//		return err
//	}
//	defer c.Close()
//
// # Lifecycle
//
// [New] creates the SDK client and logs in with Universal Auth; a failed login
// is returned as an error. [Client.Fetch] maps a folder to /<folder>, maps the
// root and default folders to /, and reads the secret in the configured
// environment. Close releases the client.
//
// # Common mistakes
//
//   - Leaving Environment empty; Infisical requires it.
//   - Expecting folders to nest; the model has exactly one level.
//   - Treating the root and default folders as distinct; both map to /.
//
// # Test recipe
//
// Use confloader's fake client for unit tests; this adapter needs a live
// Infisical instance.
package infisical
