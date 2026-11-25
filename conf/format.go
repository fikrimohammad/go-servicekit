package conf

import confcore "github.com/fikrimohammad/go-servicekit/conf/internal/core"

// Format identifies a value encoding.
type Format confcore.Format

const (
	// FormatYAML parses YAML documents and plain scalars.
	FormatYAML Format = Format(confcore.FormatYAML)
	// FormatJSON parses strict JSON.
	FormatJSON Format = Format(confcore.FormatJSON)
	// FormatTOML parses TOML.
	FormatTOML Format = Format(confcore.FormatTOML)
	// FormatScalar parses one raw value into the target type without a
	// document parser. It is valid for etcd leaves only.
	FormatScalar Format = Format(confcore.FormatScalar)
)

// Validate reports whether f is a known format.
func (f Format) Validate() error { return confcore.Format(f).Validate() }
