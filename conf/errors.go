package conf

import "errors"

var (
	// ErrNotSet reports that a config key is absent.
	ErrNotSet = errors.New("conf: key not set")
	// ErrClosed reports that the loader has been closed.
	ErrClosed = errors.New("conf: loader closed")
	// ErrUnbound reports that a Getter was not bound by a loader.
	ErrUnbound = errors.New("conf: getter is not bound")
	// ErrInvalidConfig reports an unusable loader or provider config.
	ErrInvalidConfig = errors.New("conf: invalid config")
	// ErrInvalidTag reports a malformed conf struct tag.
	ErrInvalidTag = errors.New("conf: invalid conf tag")
	// ErrInvalidType reports a config type the loader cannot handle.
	ErrInvalidType = errors.New("conf: invalid type")
	// ErrDuplicateKey reports two fields that resolve to the same key.
	ErrDuplicateKey = errors.New("conf: duplicate key")
	// ErrMissingKey reports a key that is absent at boot.
	ErrMissingKey = errors.New("conf: missing key")
	// ErrUnknownKey reports a file key with no matching struct field.
	ErrUnknownKey = errors.New("conf: unknown key")
	// ErrParse reports a value that does not decode into its target type.
	ErrParse = errors.New("conf: parse failed")
	// ErrConnect reports a provider connection failure.
	ErrConnect = errors.New("conf: connect failed")
	// ErrInvalidFormat reports an unknown value format.
	ErrInvalidFormat = errors.New("conf: invalid format")
	// ErrMissingFormat reports a leaf without a mandatory format tag.
	ErrMissingFormat = errors.New("conf: missing format")
)
