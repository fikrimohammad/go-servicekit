package etcd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fikrimohammad/go-servicekit/conf/internal/core"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// Provider reads and watches one etcd namespace. Call Load once, then Watch at
// most once, then Close. Lifecycle calls must not race with each other.
type Provider struct {
	mu     sync.Mutex
	cfg    Config
	cli    *clientv3.Client
	prefix string
	loaded bool

	closeOnce sync.Once
	closeErr  error

	watchCancel context.CancelFunc
	watchDone   chan struct{}
}

// New builds an etcd provider.
func New(cfg Config) *Provider { return &Provider{cfg: cfg} }

// Source returns "etcd".
func (p *Provider) Source() string { return "etcd" }

// Load connects to etcd, reads the namespace prefix once, and returns the
// requested keys plus the header revision.
func (p *Provider) Load(ctx context.Context, keys []string) (map[string]string, int64, error) {
	cfg := p.cfg.SetDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, 0, err
	}

	p.mu.Lock()
	if p.loaded {
		p.mu.Unlock()
		return nil, 0, fmt.Errorf("%w: Load called twice", core.ErrInvalidConfig)
	}
	p.mu.Unlock()

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{cfg.Endpoint},
		Username:    cfg.Username,
		Password:    cfg.Password,
		DialTimeout: cfg.DialTimeout,
		TLS:         cfg.TLS,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %v", core.ErrConnect, err)
	}

	prefix := cfg.Namespace + "/"
	resp, err := cli.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		_ = cli.Close()
		return nil, 0, fmt.Errorf("%w: %v", core.ErrConnect, err)
	}

	wanted := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		wanted[key] = struct{}{}
	}
	values := make(map[string]string, len(keys))
	for _, kv := range resp.Kvs {
		key := keyFromPrefix(prefix, string(kv.Key))
		if _, ok := wanted[key]; !ok {
			continue
		}
		values[key] = string(kv.Value)
	}

	p.mu.Lock()
	p.cli = cli
	p.prefix = prefix
	p.loaded = true
	p.mu.Unlock()

	return values, resp.Header.Revision, nil
}

// Watch streams changes after rev. It owns reconnect, backoff, progress
// notifications, compaction, and resync.
func (p *Provider) Watch(ctx context.Context, rev int64) (<-chan core.Event, error) {
	p.mu.Lock()
	cli := p.cli
	prefix := p.prefix
	if cli == nil {
		p.mu.Unlock()
		return nil, fmt.Errorf("%w: Watch before Load", core.ErrInvalidConfig)
	}
	if p.watchDone != nil {
		p.mu.Unlock()
		return nil, fmt.Errorf("%w: Watch called twice", core.ErrInvalidConfig)
	}
	watchCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	p.watchCancel = cancel
	p.watchDone = done
	p.mu.Unlock()

	ch := make(chan core.Event, 16)
	go p.watch(watchCtx, cli, prefix, rev, ch, done)
	return ch, nil
}

// Close stops the watch and closes the client. It is idempotent.
func (p *Provider) Close() error {
	p.closeOnce.Do(func() {
		p.mu.Lock()
		cancel := p.watchCancel
		done := p.watchDone
		p.mu.Unlock()
		// The provider owns the watch lifetime: Close must stop it without
		// relying on the caller to cancel the Watch context.
		if cancel != nil {
			cancel()
		}
		if done != nil {
			<-done
		}

		p.mu.Lock()
		cli := p.cli
		p.mu.Unlock()
		if cli != nil {
			p.closeErr = cli.Close()
		}
	})
	return p.closeErr
}

func (p *Provider) watch(ctx context.Context, cli *clientv3.Client, prefix string, rev int64, ch chan<- core.Event, done chan<- struct{}) {
	defer close(done)
	defer close(ch)

	const maxBackoff = 30 * time.Second
	backoff := time.Second

	send := func(ev core.Event) bool {
		select {
		case ch <- ev:
			return true
		case <-ctx.Done():
			return false
		}
	}

	for {
		if ctx.Err() != nil {
			return
		}

		wch := cli.Watch(ctx, prefix,
			clientv3.WithPrefix(),
			clientv3.WithRev(rev+1),
			clientv3.WithProgressNotify(),
		)

		restarted := false
		for resp := range wch {
			if resp.Canceled {
				if resp.CompactRevision > 0 || errors.Is(resp.Err(), rpctypes.ErrCompacted) {
					values, newRev, err := p.loadPrefix(ctx, cli, prefix)
					if err != nil {
						if !send(core.Event{Err: err}) {
							return
						}
					} else {
						rev = newRev
						backoff = time.Second
						if !send(core.Event{Reset: values, Revision: newRev}) {
							return
						}
					}
				} else if !send(core.Event{Err: resp.Err()}) {
					return
				}
				restarted = true
				break
			}

			if resp.IsProgressNotify() {
				rev = resp.Header.Revision
				backoff = time.Second
				if !send(core.Event{Heartbeat: true, Revision: rev}) {
					return
				}
				continue
			}

			for _, ev := range resp.Events {
				key := keyFromPrefix(prefix, string(ev.Kv.Key))
				if ev.Type == clientv3.EventTypeDelete {
					if !send(core.Event{Key: key, Deleted: true, Revision: ev.Kv.ModRevision}) {
						return
					}
				} else {
					if !send(core.Event{Key: key, Value: string(ev.Kv.Value), Revision: ev.Kv.ModRevision}) {
						return
					}
				}
				rev = ev.Kv.ModRevision
				backoff = time.Second
			}
		}

		if ctx.Err() != nil {
			return
		}

		if !restarted {
			values, newRev, err := p.loadPrefix(ctx, cli, prefix)
			if err != nil {
				if !send(core.Event{Err: err}) {
					return
				}
			} else {
				rev = newRev
				if !send(core.Event{Reset: values, Revision: newRev}) {
					return
				}
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < maxBackoff {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

func (p *Provider) loadPrefix(ctx context.Context, cli *clientv3.Client, prefix string) (map[string]string, int64, error) {
	resp, err := cli.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %v", core.ErrConnect, err)
	}
	values := make(map[string]string, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		values[keyFromPrefix(prefix, string(kv.Key))] = string(kv.Value)
	}
	return values, resp.Header.Revision, nil
}

// keyFromPrefix strips the namespace prefix from an etcd key.
func keyFromPrefix(prefix, key string) string {
	return strings.TrimPrefix(key, prefix)
}

var (
	_ core.Provider = (*Provider)(nil)
	_ core.Watcher  = (*Provider)(nil)
)
