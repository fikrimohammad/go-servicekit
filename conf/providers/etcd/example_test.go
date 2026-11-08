package etcd_test

import "github.com/fikrimohammad/go-servicekit/conf/providers/etcd"

func ExampleNew() {
	p := etcd.New(etcd.Config{
		Endpoint:  "localhost:2379",
		Namespace: "disburseapp",
	})
	_ = p
	// In a real program: loader, err := conf.New[Hot](ctx, p)
}
