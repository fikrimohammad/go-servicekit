package conf

import (
	"context"
	"maps"
	"time"

	confcore "github.com/fikrimohammad/go-servicekit/conf/internal/core"
)

func (c *loaderCore) consume(ctx context.Context, ch <-chan confcore.Event) {
	defer close(c.done)
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				c.setProviderDegraded(true)
				return
			}
			c.apply(ctx, ev)
		}
	}
}

func (c *loaderCore) apply(ctx context.Context, ev confcore.Event) {
	switch {
	case ev.Err != nil:
		c.setProviderDegraded(true)
		c.recordReload(ctx, ev.Revision, time.Now(), ev.Err, "watch")
		c.loggerClient().Warn(ctx, "conf: watch disconnected", "conf.source", c.source, "error.type", "watch")
	case ev.Heartbeat:
		c.markProviderLive(ctx)
		c.recordHeartbeat(ctx)
	case ev.Reset != nil:
		c.applyReset(ctx, ev)
	case ev.Deleted:
		c.applyDelete(ctx, ev.Key, ev.Revision)
	default:
		c.applyPut(ctx, ev.Key, ev.Value, ev.Revision)
	}
}

func (c *loaderCore) applyPut(ctx context.Context, key, raw string, rev int64) {
	start := time.Now()
	// Any event proves the watch is alive, even for a key this loader does not
	// own.
	c.markProviderLive(ctx)
	l, ok := c.leaf(key)
	if !ok {
		return
	}
	v, err := parseLeaf(c.formatter, raw, l)
	if err != nil {
		c.markDegraded(key)
		c.recordReload(ctx, rev, start, err, "parse")
		c.loggerClient().Warn(ctx, "conf: reload failed", "conf.source", c.source, "key", key, "error.type", "parse")
		return
	}
	c.update(func(values map[string]any) { values[key] = v })
	c.clearDegraded(key)
	c.recordReload(ctx, rev, start, nil, "")
}

func (c *loaderCore) applyDelete(ctx context.Context, key string, rev int64) {
	start := time.Now()
	c.markProviderLive(ctx)
	if _, ok := c.leaf(key); !ok {
		return
	}
	c.update(func(values map[string]any) { delete(values, key) })
	c.clearDegraded(key)
	c.recordReload(ctx, rev, start, nil, "")
}

func (c *loaderCore) applyReset(ctx context.Context, ev confcore.Event) {
	start := time.Now()
	old := c.snapshot.Load()
	next := make(map[string]any, len(c.leaves))
	var (
		parseErr error
		badKey   string
	)
	for key, l := range c.leaves {
		raw, ok := ev.Reset[key]
		if !ok {
			c.clearDegraded(key)
			continue
		}
		v, err := parseLeaf(c.formatter, raw, l)
		if err != nil {
			if parseErr == nil {
				parseErr, badKey = err, key
			}
			if old != nil {
				if prev, ok := old.values[key]; ok {
					next[key] = prev
				}
			}
			c.markDegraded(key)
			continue
		}
		next[key] = v
		c.clearDegraded(key)
	}
	c.snapshot.Store(&snapshot{values: next})
	c.markProviderLive(ctx)

	class := ""
	if parseErr != nil {
		class = "parse"
		c.loggerClient().Warn(ctx, "conf: reload failed", "conf.source", c.source, "key", badKey, "error.type", "parse")
	}
	c.recordReload(ctx, ev.Revision, start, parseErr, class)
}

func (c *loaderCore) update(fn func(map[string]any)) {
	s := c.snapshot.Load()
	if s == nil {
		return
	}
	next := make(map[string]any, len(s.values)+1)
	maps.Copy(next, s.values)
	fn(next)
	c.snapshot.Store(&snapshot{values: next})
}

func (c *loaderCore) leaf(key string) (leaf, bool) {
	l, ok := c.leaves[key]
	return l, ok
}

// markProviderLive clears the provider-level degradation after an event proves
// the watch is alive, and logs the recovery once.
func (c *loaderCore) markProviderLive(ctx context.Context) {
	if c.clearProviderDegraded() {
		c.loggerClient().Info(ctx, "conf: watch recovered", "conf.source", c.source)
	}
}
