package conf_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/fikrimohammad/go-servicekit/conf"
	"github.com/fikrimohammad/go-servicekit/conf/providers/etcd"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type hot struct {
	Checkout struct {
		TTL        conf.Getter[time.Duration] `conf:"name=ttl,format=scalar"`
		MaxRetries conf.Getter[int]           `conf:"name=max_retry,format=scalar"`
	} `conf:"folder=checkouts"`
	Toggle conf.Getter[bool] `conf:"name=toggle_switch,format=scalar"`
}

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

	ns := fmt.Sprintf("conf-loader-test-%d", time.Now().UnixNano())
	defer func() {
		_, _ = cli.Delete(context.Background(), ns+"/", clientv3.WithPrefix())
	}()

	put := func(key, value string) {
		t.Helper()
		if _, err := cli.Put(ctx, ns+"/"+key, value); err != nil {
			t.Fatalf("Put(%s): %v", key, err)
		}
	}

	put("checkouts/ttl", "30s")
	put("checkouts/max_retry", "3")
	put("toggle_switch", "true")

	loader, err := conf.New[hot](ctx, etcd.New(etcd.Config{Endpoint: endpoint, Namespace: ns}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer func() { _ = loader.Close() }()

	data := loader.Data()
	if got := data.Checkout.TTL.GetWithDefault(0); got != 30*time.Second {
		t.Errorf("TTL = %v, want 30s", got)
	}
	if got := data.Checkout.MaxRetries.GetWithDefault(0); got != 3 {
		t.Errorf("MaxRetries = %d, want 3", got)
	}
	if got := data.Toggle.GetWithDefault(false); !got {
		t.Error("Toggle = false, want true")
	}
	if got := loader.State(); got != conf.StateLive {
		t.Errorf("State() = %q, want %q", got, conf.StateLive)
	}

	put("toggle_switch", "false")
	waitFor(t, func() bool { return !loader.Data().Toggle.GetWithDefault(true) })

	if _, err := cli.Delete(ctx, ns+"/toggle_switch"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	waitFor(t, func() bool {
		_, err := loader.Data().Toggle.Get()
		return errors.Is(err, conf.ErrNotSet)
	})

	put("checkouts/ttl", "not-a-duration")
	waitFor(t, func() bool { return loader.State() == conf.StateDegraded })
	if got := loader.Data().Checkout.TTL.GetWithDefault(0); got != 30*time.Second {
		t.Errorf("TTL after bad value = %v, want 30s", got)
	}

	put("checkouts/ttl", "45s")
	waitFor(t, func() bool { return loader.State() == conf.StateLive })
	if got := loader.Data().Checkout.TTL.GetWithDefault(0); got != 45*time.Second {
		t.Errorf("TTL after recovery = %v, want 45s", got)
	}

	if err := loader.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := loader.Data().Checkout.TTL.Get(); !errors.Is(err, conf.ErrClosed) {
		t.Errorf("Get after Close = %v, want ErrClosed", err)
	}
	if got := loader.State(); got != conf.StateClosed {
		t.Errorf("State after Close = %q, want %q", got, conf.StateClosed)
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met before deadline")
}
