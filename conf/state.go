package conf

// State reports the loader's health and lifecycle.
type State string

const (
	// StateLive reports that the last load or update succeeded.
	StateLive State = "live"
	// StateDegraded reports that a key kept last-known-good after a failure,
	// or that the provider's watch is disconnected.
	StateDegraded State = "degraded"
	// StateClosed reports that Close ran. It is terminal.
	StateClosed State = "closed"
)

// String returns the state's lowercase name.
func (s State) String() string { return string(s) }

func (c *loaderCore) markDegraded(key string) {
	c.stateMu.Lock()
	c.degraded[key] = struct{}{}
	c.recomputeLocked()
	c.stateMu.Unlock()
}

func (c *loaderCore) clearDegraded(key string) {
	c.stateMu.Lock()
	delete(c.degraded, key)
	c.recomputeLocked()
	c.stateMu.Unlock()
}

func (c *loaderCore) setProviderDegraded(degraded bool) {
	c.stateMu.Lock()
	c.providerDegraded = degraded
	c.recomputeLocked()
	c.stateMu.Unlock()
}

func (c *loaderCore) clearProviderDegraded() bool {
	c.stateMu.Lock()
	was := c.providerDegraded
	c.providerDegraded = false
	c.recomputeLocked()
	c.stateMu.Unlock()
	return was
}

func (c *loaderCore) recomputeLocked() {
	if c.closed.Load() {
		c.state = StateClosed
		return
	}
	if c.providerDegraded || len(c.degraded) > 0 {
		c.state = StateDegraded
		return
	}
	c.state = StateLive
}
