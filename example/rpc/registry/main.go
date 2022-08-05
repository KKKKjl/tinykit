package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/KKKKjl/tinykit/internal/registry"
	"github.com/KKKKjl/tinykit/internal/registry/etcd"
)

func main() {
	client, err := etcd.NewEtcdClient()
	if err != nil {
		panic(err)
	}
	defer client.Close()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	client.Register(&registry.Service{
		Name: "node1",
		Addr: "http://localhost:8090",
	}, 10)

	// client.Register(&registry.Service{
	// 	Name: "node2",
	// 	Addr: "http://localhost:8086",
	// }, 10)

	// client.Register(&registry.Service{
	// 	Name: "node3",
	// 	Addr: "http://localhost:8087",
	// }, 10)

	select {
	case <-interrupt:
		log.Println("interrupt")
		return
	}
}
