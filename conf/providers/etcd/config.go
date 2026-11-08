package etcd

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/fikrimohammad/go-servicekit/conf"
)

// Config configures the etcd provider. The value format is not here: every
// conf leaf declares its own format.
type Config struct {
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
func (c Config) SetDefaults() Config {
	if c.DialTimeout <= 0 {
		c.DialTimeout = 5 * time.Second
	}
	return c
}

// Validate reports whether the config is usable.
func (c Config) Validate() error {
	var errs []error
	if _, _, err := net.SplitHostPort(c.Endpoint); err != nil {
		errs = append(errs, fmt.Errorf("%w: endpoint must be host:port", conf.ErrInvalidConfig))
	}
	if c.Namespace == "" ||
		strings.HasPrefix(c.Namespace, "/") ||
		strings.HasSuffix(c.Namespace, "/") ||
		strings.Contains(c.Namespace, "//") {
		errs = append(errs, fmt.Errorf("%w: namespace must be non-empty without slashes", conf.ErrInvalidConfig))
	}
	if (c.Username == "") != (c.Password == "") {
		errs = append(errs, fmt.Errorf("%w: username and password must be set together", conf.ErrInvalidConfig))
	}
	return errors.Join(errs...)
}
