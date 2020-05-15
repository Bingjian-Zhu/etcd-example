package main

import (
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"

	"etcd-example/4-etcd-grpclb/etcdv3"
	pb "etcd-example/4-etcd-grpclb/proto"
)

// Address 连接地址
const Address string = ":8000"

var grpcClient pb.StreamServerClient

func main() {
	var endpoints = []string{"localhost:2379"}
	r := etcdv3.NewResolver(endpoints)
	resolver.Register(r)
	// 连接服务器
	conn, err := grpc.Dial(r.Scheme()+"://8.8.8.8/simple_grpc", grpc.WithBalancerName("round_robin"), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("net.Connect err: %v", err)
	}
	defer conn.Close()

	// 建立gRPC连接
	grpcClient = pb.NewStreamServerClient(conn)
	route()
}

// route 调用服务端Route方法
func route() {
	// 创建发送结构体
	req := pb.SimpleRequest{
		Data: "grpc",
	}
	// 调用我们的服务(Route方法)
	// 同时传入了一个 context.Context ，在有需要时可以让我们改变RPC的行为，比如超时/取消一个正在运行的RPC
	res, err := grpcClient.Route(context.Background(), &req)
	if err != nil {
		log.Fatalf("Call Route err: %v", err)
	}
	// 打印返回值
	log.Println(res)
}
