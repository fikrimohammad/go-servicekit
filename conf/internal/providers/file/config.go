package file

import (
	"errors"
	"fmt"

	"github.com/fikrimohammad/go-servicekit/conf/internal/core"
)

// Config configures the file provider. Both fields are required.
type Config struct {
	// Path is the config file path.
	Path string
	// Format is the document format: YAML, JSON, or TOML.
	Format core.Format
}

// SetDefaults returns a copy with zero values filled in. The file provider has
// no defaults; both fields are required.
func (c Config) SetDefaults() Config { return c }

// Validate reports whether the config is usable.
func (c Config) Validate() error {
	var errs []error
	if c.Path == "" {
		errs = append(errs, fmt.Errorf("%w: path is required", core.ErrInvalidConfig))
	}
	switch c.Format {
	case core.FormatYAML, core.FormatJSON, core.FormatTOML:
	default:
		errs = append(errs, fmt.Errorf("%w: %q", core.ErrInvalidFormat, c.Format))
	}
	return errors.Join(errs...)
}
