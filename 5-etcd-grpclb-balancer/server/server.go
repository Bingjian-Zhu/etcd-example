package main

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"

	"etcd-example/5-etcd-grpclb-balancer/etcdv3"
	pb "etcd-example/5-etcd-grpclb-balancer/proto"
)

// SimpleService 定义我们的服务
type SimpleService struct{}

const (
	// Address 监听地址
	Address string = "localhost:8000"
	// Network 网络通信协议
	Network string = "tcp"
	// SerName 服务名称
	SerName string = "simple_grpc"
)

// EtcdEndpoints etcd地址
var EtcdEndpoints = []string{"localhost:2379"}

func main() {
	// 监听本地端口
	listener, err := net.Listen(Network, Address)
	if err != nil {
		log.Fatalf("net.Listen err: %v", err)
	}
	log.Println(Address + " net.Listing...")
	// 新建gRPC服务器实例
	grpcServer := grpc.NewServer()
	// 在gRPC服务器注册我们的服务
	pb.RegisterSimpleServer(grpcServer, &SimpleService{})
	//把服务注册到etcd
	ser, err := etcdv3.NewServiceRegister(EtcdEndpoints, SerName+"/"+Address, "1", 5)
	if err != nil {
		log.Fatalf("register service err: %v", err)
	}
	defer ser.Close()
	//用服务器 Serve() 方法以及我们的端口信息区实现阻塞等待，直到进程被杀死或者 Stop() 被调用
	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatalf("grpcServer.Serve err: %v", err)
	}
}

// Route 实现Route方法
func (s *SimpleService) Route(ctx context.Context, req *pb.SimpleRequest) (*pb.SimpleResponse, error) {
	log.Println("receive: " + req.Data)
	res := pb.SimpleResponse{
		Code:  200,
		Value: "hello " + req.Data,
	}
	return &res, nil
}
