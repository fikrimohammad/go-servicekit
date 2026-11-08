package file_test

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/fikrimohammad/go-servicekit/conf"
	"github.com/fikrimohammad/go-servicekit/conf/providers/file"
)

func ExampleProvider() {
	p := file.New(file.Config{Path: "testdata/app.yaml", Format: conf.FormatYAML})
	values, _, err := p.Load(context.Background(), []string{"checkouts/ttl"})
	if err != nil {
		panic(err)
	}

	got, err := p.Parse(values["checkouts/ttl"], reflect.TypeFor[time.Duration]())
	if err != nil {
		panic(err)
	}
	fmt.Println(got)

	// Output:
	// 30s
}
