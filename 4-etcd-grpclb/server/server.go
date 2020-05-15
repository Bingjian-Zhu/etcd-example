package main

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"

	pb "etcd-example/4-etcd-grpclb/proto"
)

// SimpleService 定义我们的服务
type SimpleService struct{}

const (
	// Address 监听地址
	Address string = "127.0.0.1:8000"
	// Network 网络通信协议
	Network string = "tcp"
	// svcName 服务名称
	svcName string = "simple_grpc"
)

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
	go etcdv3.NewServiceRegister([]string{"localhost:2379"}, svcName, Address, 5)
	//用服务器 Serve() 方法以及我们的端口信息区实现阻塞等待，直到进程被杀死或者 Stop() 被调用
	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatalf("grpcServer.Serve err: %v", err)
	}
}

// Route 实现Route方法
func (s *SimpleService) Route(ctx context.Context, req *pb.SimpleRequest) (*pb.SimpleResponse, error) {
	res := pb.SimpleResponse{
		Code:  200,
		Value: "hello " + req.Data,
	}
	return &res, nil
}
