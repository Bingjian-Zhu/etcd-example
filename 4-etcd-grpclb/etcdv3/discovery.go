package etcdv3

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/coreos/etcd/mvcc/mvccpb"
	"go.etcd.io/etcd/clientv3"
	"google.golang.org/grpc/resolver"
)

const schema = "grpclb"

//ServiceDiscovery 服务发现
type ServiceDiscovery struct {
	cli *clientv3.Client //etcd client
	cc  resolver.ClientConn
}

//NewServiceDiscovery  新建发现服务
func NewServiceDiscovery(endpoints []string) resolver.Builder {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}

	return &ServiceDiscovery{
		cli: cli,
	}
}

//Build 创建一个解析器，该解析器将用于监视名称解析更新
func (s *ServiceDiscovery) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOption) (resolver.Resolver, error) {
	s.cc = cc
	prefix := "/" + target.Scheme + "/" + target.Endpoint + "/"
	var addrList []resolver.Address
	//根据前缀获取现有的key
	resp, err := s.cli.Get(context.Background(), prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	for _, ev := range resp.Kvs {
		addrList = append(addrList, resolver.Address{Addr: strings.TrimPrefix(string(ev.Key), prefix)})
	}
	s.cc.NewAddress(addrList)
	//监视前缀，修改变更的server
	go s.watcher(prefix, addrList)
	return s, nil
}

// ResolveNow new
func (s *ServiceDiscovery) ResolveNow(rn resolver.ResolveNowOption) {
	log.Println("ResolveNow") // TODO check
}

//Scheme return schema
func (s *ServiceDiscovery) Scheme() string {
	return schema
}

//Close 关闭服务
func (s *ServiceDiscovery) Close() {
	s.cli.Close()
}

//watcher 监听前缀
func (s *ServiceDiscovery) watcher(prefix string, addrList []resolver.Address) {
	rch := s.cli.Watch(context.Background(), prefix, clientv3.WithPrefix())
	log.Printf("watching prefix:%s now...", prefix)
	for wresp := range rch {
		for _, ev := range wresp.Events {
			addr := strings.TrimPrefix(string(ev.Kv.Key), prefix)
			switch ev.Type {
			case mvccpb.PUT: //新增
				if !exist(addrList, addr) {
					addrList = append(addrList, resolver.Address{Addr: addr})
					s.cc.NewAddress(addrList)
				}
			case mvccpb.DELETE: //删除
				if tmp, ok := remove(addrList, addr); ok {
					addrList = tmp
					s.cc.NewAddress(addrList)
				}
			}
		}
	}
}

func exist(l []resolver.Address, addr string) bool {
	for i := range l {
		if l[i].Addr == addr {
			return true
		}
	}
	return false
}

func remove(s []resolver.Address, addr string) ([]resolver.Address, bool) {
	for i := range s {
		if s[i].Addr == addr {
			s[i] = s[len(s)-1]
			return s[:len(s)-1], true
		}
	}
	return nil, false
}
