package core

import (
	"context"
	"reflect"
)

// Provider is the storage contract. A Loader binds one Provider, calls Load
// once, and owns the Provider until Close. A Provider must not be shared
// between Loaders.
type Provider interface {
	// Source returns a stable, low-cardinality telemetry label: "file" or
	// "etcd" for the built-in providers.
	Source() string
	// Load returns the raw value for each requested key. Missing keys are
	// absent from the map. rev is opaque to the Loader and is passed back to
	// Watch. Load is called once per Loader.
	Load(ctx context.Context, keys []string) (map[string]string, int64, error)
	// Close releases resources. It is idempotent and safe before a successful
	// Load.
	Close() error
}

// Watcher is a Provider that streams changes. A Loader calls Watch at most
// once, after Load.
type Watcher interface {
	// Watch streams changes after rev. The channel closes only when ctx is
	// done. The provider owns reconnect and resync. Watch must return promptly;
	// its ctx is detached from the boot context and canceled by Close.
	Watch(ctx context.Context, rev int64) (<-chan Event, error)
}

// Formatter is a Provider that owns parsing for its values. The loader calls
// Parse for every boot value and every watched update, and rejects per-leaf
// format tags. The file provider implements it; the etcd provider does not.
type Formatter interface {
	// Parse decodes raw into a value of type t. The result must be assignable
	// to t; the loader converts it to t. nil is allowed only when t can be
	// nil.
	Parse(raw string, t reflect.Type) (any, error)
}

// Event is one change from a Watcher. A Watcher must set exactly one kind of
// change per event: Err, Heartbeat, Reset, Deleted, or a put. If more than one
// is set, only the first in that order is applied and the others are ignored.
// A Watcher must not mutate an Event or its Reset map after sending it.
type Event struct {
	// Key is the config key for a put or delete.
	Key string
	// Value is the raw value for a put.
	Value string
	// Deleted reports that Key is gone.
	Deleted bool
	// Reset, when non-nil, replaces the whole snapshot after a resync.
	Reset map[string]string
	// Revision is the store revision, or zero if unknown.
	Revision int64
	// Heartbeat reports that the watch is alive with no value change.
	Heartbeat bool
	// Err reports that the watch is degraded. It carries no value change.
	Err error
}
