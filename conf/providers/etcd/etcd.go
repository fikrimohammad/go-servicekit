package etcd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fikrimohammad/go-servicekit/conf"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// Provider reads and watches one etcd namespace.
type Provider struct {
	mu     sync.Mutex
	cfg    Config
	cli    *clientv3.Client
	prefix string
	loaded bool

	closeOnce sync.Once
	closeErr  error

	watchDone chan struct{}
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
		return nil, 0, fmt.Errorf("%w: Load called twice", conf.ErrInvalidConfig)
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
		return nil, 0, fmt.Errorf("%w: %v", conf.ErrConnect, err)
	}

	prefix := cfg.Namespace + "/"
	resp, err := cli.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		_ = cli.Close()
		return nil, 0, fmt.Errorf("%w: %v", conf.ErrConnect, err)
	}

	wanted := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		wanted[key] = struct{}{}
	}
	values := make(map[string]string, len(keys))
	for _, kv := range resp.Kvs {
		key := keyFromPrefix(prefix, string(kv.Key))
		if len(wanted) > 0 {
			if _, ok := wanted[key]; !ok {
				continue
			}
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
func (p *Provider) Watch(ctx context.Context, rev int64) (<-chan conf.Event, error) {
	p.mu.Lock()
	cli := p.cli
	prefix := p.prefix
	if cli == nil {
		p.mu.Unlock()
		return nil, fmt.Errorf("%w: Watch before Load", conf.ErrInvalidConfig)
	}
	if p.watchDone != nil {
		p.mu.Unlock()
		return nil, fmt.Errorf("%w: Watch called twice", conf.ErrInvalidConfig)
	}
	done := make(chan struct{})
	p.watchDone = done
	p.mu.Unlock()

	ch := make(chan conf.Event, 16)
	go p.watch(ctx, cli, prefix, rev, ch, done)
	return ch, nil
}

// Close stops the watch and closes the client. It is idempotent.
func (p *Provider) Close() error {
	p.closeOnce.Do(func() {
		p.mu.Lock()
		done := p.watchDone
		p.mu.Unlock()
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

func (p *Provider) watch(ctx context.Context, cli *clientv3.Client, prefix string, rev int64, ch chan<- conf.Event, done chan<- struct{}) {
	defer close(done)
	defer close(ch)

	const maxBackoff = 30 * time.Second
	backoff := time.Second

	send := func(ev conf.Event) bool {
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
						if !send(conf.Event{Err: err}) {
							return
						}
					} else {
						rev = newRev
						backoff = time.Second
						if !send(conf.Event{Reset: values, Revision: newRev}) {
							return
						}
					}
				} else if !send(conf.Event{Err: resp.Err()}) {
					return
				}
				restarted = true
				break
			}

			if resp.IsProgressNotify() {
				rev = resp.Header.Revision
				backoff = time.Second
				if !send(conf.Event{Heartbeat: true, Revision: rev}) {
					return
				}
				continue
			}

			for _, ev := range resp.Events {
				key := keyFromPrefix(prefix, string(ev.Kv.Key))
				if ev.Type == clientv3.EventTypeDelete {
					if !send(conf.Event{Key: key, Deleted: true, Revision: ev.Kv.ModRevision}) {
						return
					}
				} else {
					if !send(conf.Event{Key: key, Value: string(ev.Kv.Value), Revision: ev.Kv.ModRevision}) {
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
				if !send(conf.Event{Err: err}) {
					return
				}
			} else {
				rev = newRev
				if !send(conf.Event{Reset: values, Revision: newRev}) {
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
		return nil, 0, fmt.Errorf("%w: %v", conf.ErrConnect, err)
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
	_ conf.Provider = (*Provider)(nil)
	_ conf.Watcher  = (*Provider)(nil)
)
