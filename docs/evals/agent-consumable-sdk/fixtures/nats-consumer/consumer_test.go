package eval

import (
	"context"
	"errors"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"

	"github.com/fikrimohammad/go-servicekit/identity"
	"github.com/fikrimohammad/go-servicekit/nats/consumer"
	"github.com/fikrimohammad/go-servicekit/nats/producer"
)

// startNATS starts an in-process JetStream server and stops it when the test
// ends.
func startNATS(t *testing.T) *natsserver.Server {
	t.Helper()
	ns, err := natsserver.NewServer(&natsserver.Options{
		Port:      -1,
		JetStream: true,
		StoreDir:  t.TempDir(),
	})
	if err != nil {
		t.Fatalf("natsserver.NewServer: %v", err)
	}
	t.Cleanup(func() {
		ns.Shutdown()
		ns.WaitForShutdown()
	})
	go ns.Start()
	if !ns.ReadyForConnections(10 * time.Second) {
		t.Fatal("NATS server not ready")
	}
	return ns
}

// consume starts a consumer for topic/group, calls h for each message, and
// returns nil after the first message. If ctx ends first it returns ctx.Err().
//
// TODO: build the consumer with the nats/consumer package, register h, start,
// wait for one message, and shut the consumer down.
func consume(_ context.Context, _, _, _ string, _ consumer.HandlerFunc) error {
	_ = identity.Info{}
	_ = consumer.Config{}
	_ = producer.Message{}
	return errors.New("not implemented")
}

func TestConsume(t *testing.T) {
	ns := startNATS(t)
	url := ns.ClientURL()

	p, err := producer.New(identity.Info{Name: "eval"})
	if err != nil {
		t.Fatalf("producer.New: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := p.Register(producer.Config{Endpoints: []string{url}, Topic: "orders"}); err != nil {
		t.Fatalf("producer.Register: %v", err)
	}
	if err := p.Start(ctx); err != nil {
		t.Fatalf("producer.Start: %v", err)
	}
	t.Cleanup(func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		_ = p.Shutdown(shutdownCtx)
	})

	// Publish before the consumer starts; ConsumeFromFirst must still deliver it.
	if err := p.PublishSync(ctx, producer.Message{
		Topic: "orders", Tag: "created", Key: "order-1", Body: []byte("hello"),
	}); err != nil {
		t.Fatalf("PublishSync: %v", err)
	}

	got := make(chan string, 1)
	err = consume(ctx, url, "orders", "eval-group", func(_ context.Context, msg consumer.Message) error {
		got <- string(msg.Body())
		return nil
	})
	if err != nil {
		t.Fatalf("consume: %v", err)
	}

	select {
	case v := <-got:
		if v != "hello" {
			t.Errorf("body = %q, want hello", v)
		}
	case <-ctx.Done():
		t.Fatal("consume returned without delivering a message")
	}
}
