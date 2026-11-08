package etcd_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/fikrimohammad/go-servicekit/conf"
	"github.com/fikrimohammad/go-servicekit/conf/providers/etcd"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func TestEtcdIntegration(t *testing.T) {
	endpoint := os.Getenv("CONF_ETCD_ENDPOINT")
	if endpoint == "" {
		t.Skip("CONF_ETCD_ENDPOINT is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{endpoint},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("clientv3.New: %v", err)
	}
	defer cli.Close()

	ns := fmt.Sprintf("conf-test-%d", time.Now().UnixNano())
	defer func() {
		_, _ = cli.Delete(context.Background(), ns+"/", clientv3.WithPrefix())
	}()

	put := func(key, value string) int64 {
		t.Helper()
		resp, err := cli.Put(ctx, ns+"/"+key, value)
		if err != nil {
			t.Fatalf("Put(%s): %v", key, err)
		}
		return resp.Header.Revision
	}

	put("checkouts/ttl", "30s")
	put("toggle_switch", "true")

	p := etcd.New(etcd.Config{Endpoint: endpoint, Namespace: ns})
	values, rev, err := p.Load(ctx, []string{"checkouts/ttl", "toggle_switch"})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if values["checkouts/ttl"] != "30s" {
		t.Errorf("Load(ttl) = %q, want 30s", values["checkouts/ttl"])
	}

	watchCtx, watchCancel := context.WithCancel(ctx)
	defer p.Close()
	defer watchCancel()

	events, err := p.Watch(watchCtx, rev)
	if err != nil {
		t.Fatalf("Watch: %v", err)
	}

	put("toggle_switch", "false")
	ev := awaitEvent(t, events, func(ev conf.Event) bool {
		return ev.Key == "toggle_switch" && !ev.Deleted && ev.Value == "false"
	})
	if ev.Revision == 0 {
		t.Error("put event revision = 0, want non-zero")
	}

	if _, err := cli.Delete(ctx, ns+"/toggle_switch"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	awaitEvent(t, events, func(ev conf.Event) bool {
		return ev.Key == "toggle_switch" && ev.Deleted
	})
}

func TestEtcdIntegrationCompaction(t *testing.T) {
	endpoint := os.Getenv("CONF_ETCD_ENDPOINT")
	if endpoint == "" {
		t.Skip("CONF_ETCD_ENDPOINT is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{endpoint},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("clientv3.New: %v", err)
	}
	defer cli.Close()

	ns := fmt.Sprintf("conf-test-compact-%d", time.Now().UnixNano())
	defer func() {
		_, _ = cli.Delete(context.Background(), ns+"/", clientv3.WithPrefix())
	}()

	put := func(key, value string) int64 {
		t.Helper()
		resp, err := cli.Put(ctx, ns+"/"+key, value)
		if err != nil {
			t.Fatalf("Put(%s): %v", key, err)
		}
		return resp.Header.Revision
	}

	put("checkouts/ttl", "30s")
	put("toggle_switch", "true")

	p := etcd.New(etcd.Config{Endpoint: endpoint, Namespace: ns})
	_, rev, err := p.Load(ctx, []string{"checkouts/ttl", "toggle_switch"})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	put("toggle_switch", "false")
	next := put("checkouts/max_retry", "5")
	if _, err := cli.Compact(ctx, next); err != nil {
		t.Fatalf("Compact(%d): %v", next, err)
	}

	watchCtx, watchCancel := context.WithCancel(ctx)
	defer p.Close()
	defer watchCancel()

	events, err := p.Watch(watchCtx, rev)
	if err != nil {
		t.Fatalf("Watch: %v", err)
	}

	ev := awaitEvent(t, events, func(ev conf.Event) bool { return ev.Reset != nil })
	if got := ev.Reset["toggle_switch"]; got != "false" {
		t.Errorf("reset toggle_switch = %q, want false", got)
	}
	if got := ev.Reset["checkouts/max_retry"]; got != "5" {
		t.Errorf("reset checkouts/max_retry = %q, want 5", got)
	}
}

func awaitEvent(t *testing.T, ch <-chan conf.Event, match func(conf.Event) bool) conf.Event {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				t.Fatal("watch channel closed")
			}
			if ev.Err != nil {
				t.Fatalf("watch error: %v", ev.Err)
			}
			if match(ev) {
				return ev
			}
		case <-timer.C:
			t.Fatal("timed out waiting for event")
		}
	}
}
