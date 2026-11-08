// Package file reads one YAML, JSON, or TOML configuration document.
//
// # When to use
//
// Use file for boot configuration that is loaded once at startup. A file
// provider owns its document format and implements [conf.Formatter], so it
// decodes every value with that format; leaves must not carry a format tag.
//
// # Import
//
//	import "github.com/fikrimohammad/go-servicekit/conf/providers/file"
//
// # Minimal example
//
//	p := file.New(file.Config{Path: "app.yaml", Format: conf.FormatYAML})
//	loader, err := conf.New[Boot](ctx, p)
//	if err != nil {
//		return err
//	}
//	defer loader.Close()
//
//	boot := loader.Data()
//
// # Lifecycle
//
// [Provider.Load] validates the config, reads the file once, walks the document
// by the requested keys, and returns raw values in the document format.
// [Provider.Parse] decodes those values with Config.Format. [Provider.Close] is
// a no-op. A file change requires a new process.
//
// # Common mistakes
//
//   - Omitting Format; there is no extension detection.
//   - Using FormatScalar, which is for etcd leaves only.
//   - Expecting a file change to reload the loader.
//   - Adding a format tag to a file-loaded leaf.
//
// # Test recipe
//
// Write a temp file with t.TempDir and os.WriteFile, then pass the provider to
// conf.New or call Load directly. The package example reads testdata/app.yaml.
package file
