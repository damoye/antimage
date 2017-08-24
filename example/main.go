package main

import (
	"fmt"

	etcdconfig "github.com/damoye/etcd-config"
)

func main() {
	c, err := etcdconfig.New([]string{"localhost:2379"}, "foo")
	if err != nil {
		panic(err)
	}
	fmt.Println("config bar is ", c.Get("bar"))
}
