package main

import (
	"etcd-example/3-etcd-service-discovery/register"
	"log"
	"time"
)

func main() {
	var endpoints = []string{"localhost:2379"}
	ser, err := register.NewService(endpoints, "/server/node1", "node1", 5)
	if err != nil {
		log.Fatalln(err)
	}
	//监听续租相应chan
	go ser.ListenLeaseRespChan()
	select {
	case <-time.After(20 * time.Second):
		ser.Close()
	}
}
