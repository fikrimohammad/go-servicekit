package conf

// Event is one change from a Watcher.
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
