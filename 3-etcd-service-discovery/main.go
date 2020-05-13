package main

import (
	"etcd-example/3-etcd-service-discovery/register"
	"log"
)

func main() {
	var endpoints = []string{"localhost:2379"}
	ser, err := register.NewService(endpoints, "/server/node1", "node1", 5)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(ser)
	select {}
}
