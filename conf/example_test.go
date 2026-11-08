package conf_test

import (
	"context"
	"fmt"
	"time"

	"github.com/fikrimohammad/go-servicekit/conf"
	"github.com/fikrimohammad/go-servicekit/conf/providers/etcd"
	"github.com/fikrimohammad/go-servicekit/conf/providers/file"
)

type exampleBoot struct {
	Checkout struct {
		TTL conf.Getter[time.Duration] `conf:"name=ttl"`
	} `conf:"folder=checkouts"`
}

func ExampleNew() {
	loader, err := conf.New[exampleBoot](context.Background(), file.New(file.Config{
		Path:   "testdata/app.yaml",
		Format: conf.FormatYAML,
	}))
	if err != nil {
		panic(err)
	}
	defer func() { _ = loader.Close() }()

	fmt.Println(loader.Data().Checkout.TTL.GetWithDefault(0))

	// Output:
	// 30s
}

func ExampleNew_etcd() {
	p := etcd.New(etcd.Config{
		Endpoint:  "localhost:2379",
		Namespace: "disburseapp",
	})
	_ = p
	// In a real program: loader, err := conf.New[hot](ctx, p)
}
