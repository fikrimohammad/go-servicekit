package conf_test

import (
	"context"
	"fmt"
	"time"

	"github.com/fikrimohammad/go-servicekit/conf"
)

type exampleBoot struct {
	Checkout struct {
		TTL conf.Getter[time.Duration] `conf:"name=ttl"`
	} `conf:"folder=checkouts"`
}

func ExampleNew() {
	loader, err := conf.New[exampleBoot](context.Background(), conf.File(conf.FileConfig{
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
	source := conf.Etcd(conf.EtcdConfig{
		Endpoint:  "localhost:2379",
		Namespace: "disburseapp",
	})
	_ = source
	// In a real program: loader, err := conf.New[hot](ctx, source)
}
