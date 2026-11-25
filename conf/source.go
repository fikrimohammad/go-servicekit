package conf

import (
	confcore "github.com/fikrimohammad/go-servicekit/conf/internal/core"
	"github.com/fikrimohammad/go-servicekit/conf/internal/providers/etcd"
	"github.com/fikrimohammad/go-servicekit/conf/internal/providers/file"
)

// Source describes one built-in storage backend for [New]. Create it with
// [File] or [Etcd]. A Source can create more than one Loader; each Loader gets
// its own provider.
type Source struct {
	newProvider func() confcore.Provider
	err         error
}

// File returns a source that reads one YAML, JSON, or TOML document. The config
// is validated immediately; [New] reports a validation error.
func File(cfg FileConfig) Source {
	cfg = cfg.SetDefaults()
	if err := cfg.Validate(); err != nil {
		return Source{err: err}
	}
	return Source{
		newProvider: func() confcore.Provider {
			return file.New(file.Config{Path: cfg.Path, Format: confcore.Format(cfg.Format)})
		},
	}
}

// Etcd returns a source that reads and watches one etcd namespace. The config
// is validated immediately; [New] reports a validation error.
func Etcd(cfg EtcdConfig) Source {
	cfg = cfg.SetDefaults()
	if err := cfg.Validate(); err != nil {
		return Source{err: err}
	}
	return Source{
		newProvider: func() confcore.Provider {
			return etcd.New(etcd.Config{
				Endpoint:    cfg.Endpoint,
				Username:    cfg.Username,
				Password:    cfg.Password,
				Namespace:   cfg.Namespace,
				DialTimeout: cfg.DialTimeout,
				TLS:         cfg.TLS,
			})
		},
	}
}
