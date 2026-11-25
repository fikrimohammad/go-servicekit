package conf

import confcore "github.com/fikrimohammad/go-servicekit/conf/internal/core"

var (
	// ErrNotSet reports that a config key is absent.
	ErrNotSet = confcore.ErrNotSet
	// ErrClosed reports that the loader has been closed.
	ErrClosed = confcore.ErrClosed
	// ErrUnbound reports that a Getter was not bound by a loader.
	ErrUnbound = confcore.ErrUnbound
	// ErrInvalidConfig reports an unusable loader or source config.
	ErrInvalidConfig = confcore.ErrInvalidConfig
	// ErrInvalidTag reports a malformed conf struct tag.
	ErrInvalidTag = confcore.ErrInvalidTag
	// ErrInvalidType reports a config type the loader cannot handle.
	ErrInvalidType = confcore.ErrInvalidType
	// ErrDuplicateKey reports two fields that resolve to the same key.
	ErrDuplicateKey = confcore.ErrDuplicateKey
	// ErrMissingKey reports a key that is absent at boot.
	ErrMissingKey = confcore.ErrMissingKey
	// ErrUnknownKey reports a file key with no matching struct field.
	ErrUnknownKey = confcore.ErrUnknownKey
	// ErrParse reports a value that does not decode into its target type.
	ErrParse = confcore.ErrParse
	// ErrConnect reports a source connection failure.
	ErrConnect = confcore.ErrConnect
	// ErrInvalidFormat reports an unknown value format.
	ErrInvalidFormat = confcore.ErrInvalidFormat
	// ErrMissingFormat reports a leaf without a mandatory format tag.
	ErrMissingFormat = confcore.ErrMissingFormat
)
