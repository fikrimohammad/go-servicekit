package conf

import (
	"crypto/tls"
	"time"

	confcore "github.com/fikrimohammad/go-servicekit/conf/internal/core"
	"github.com/fikrimohammad/go-servicekit/conf/internal/providers/etcd"
	"github.com/fikrimohammad/go-servicekit/conf/internal/providers/file"
)

// FileConfig configures a file source. Both fields are required.
type FileConfig struct {
	// Path is the config file path.
	Path string
	// Format is the document format: YAML, JSON, or TOML. Required.
	Format Format
}

// SetDefaults returns a copy with zero values filled in. The file source has no
// defaults; both fields are required.
func (c FileConfig) SetDefaults() FileConfig { return c }

// Validate reports whether the config is usable.
func (c FileConfig) Validate() error {
	return file.Config{Path: c.Path, Format: confcore.Format(c.Format)}.Validate()
}

// EtcdConfig configures an etcd source. The value format is not here: every
// conf leaf declares its own format.
type EtcdConfig struct {
	// Endpoint is the etcd host:port. Required.
	Endpoint string
	// Username and Password are optional; set both or neither.
	Username string
	// Password is the password paired with Username.
	Password string
	// Namespace is the key prefix, without a leading or trailing slash.
	// Required.
	Namespace string
	// DialTimeout bounds the initial connection. Defaults to 5s.
	DialTimeout time.Duration
	// TLS, when non-nil, secures the connection.
	TLS *tls.Config
}

// SetDefaults returns a copy with zero values filled in.
func (c EtcdConfig) SetDefaults() EtcdConfig {
	c.DialTimeout = etcd.Config{DialTimeout: c.DialTimeout}.SetDefaults().DialTimeout
	return c
}

// Validate reports whether the config is usable.
func (c EtcdConfig) Validate() error {
	return etcd.Config{
		Endpoint:    c.Endpoint,
		Username:    c.Username,
		Password:    c.Password,
		Namespace:   c.Namespace,
		DialTimeout: c.DialTimeout,
		TLS:         c.TLS,
	}.Validate()
}
