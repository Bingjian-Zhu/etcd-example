### etcd实现服务发现

### 前言

[etcd环境安装与使用](https://bingjian-zhu.github.io/2020/05/09/etcd%E7%8E%AF%E5%A2%83%E5%AE%89%E8%A3%85%E4%B8%8E%E4%BD%BF%E7%94%A8/)文章中介绍了etcd的安装及`v3 API`使用，本篇将介绍如何使用etcd实现服务发现功能。

### 服务发现介绍
服务发现要解决的也是分布式系统中最常见的问题之一，即在同一个分布式集群中的进程或服务，要如何才能找到对方并建立连接。本质上来说，服务发现就是想要了解集群中是否有进程在监听 udp 或 tcp 端口，并且通过名字就可以查找和连接。

![](https://img2020.cnblogs.com/blog/1508611/202005/1508611-20200514171049345-955603950.png)

服务发现需要实现一下基本功能：

* `服务注册`：同一service的所有节点注册到相同目录下，节点启动后将自己的信息注册到所属服务的目录中。

* `健康检查`：服务节点定时进行健康检查。注册到服务目录中的信息设置一个较短的TTL，运行正常的服务节点每隔一段时间会去更新信息的TTL ，从而达到健康检查效果。

* `服务发现`：通过服务节点能查询到服务提供外部访问的 IP 和端口号。比如网关代理服务时能够及时的发现服务中新增节点、丢弃不可用的服务节点。

接下来介绍如何使用etcd实现服务发现。

### 服务注册及健康检查

根据etcd的`v3 API`，当启动一个服务时候，我们把服务的地址写进etcd，注册服务。同时绑定租约（lease），并以续租约（keep leases alive）的方式检测服务是否正常运行，从而实现健康检查。

go代码实现：
```go
package main

import (
	"context"
	"log"
	"time"

	"go.etcd.io/etcd/clientv3"
)

//ServiceRegister 创建租约注册服务
type ServiceRegister struct {
	cli     *clientv3.Client //etcd client
	leaseID clientv3.LeaseID //租约ID
	//租约keepalieve相应chan
	keepAliveChan <-chan *clientv3.LeaseKeepAliveResponse
	key           string //key
	val           string //value
}

//NewServiceRegister 新建注册服务
func NewServiceRegister(endpoints []string, key, val string, lease int64) (*ServiceRegister, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}

	ser := &ServiceRegister{
		cli: cli,
		key: key,
		val: val,
	}

	//申请租约设置时间keepalive
	if err := ser.putKeyWithLease(lease); err != nil {
		return nil, err
	}

	return ser, nil
}

//设置租约
func (s *ServiceRegister) putKeyWithLease(lease int64) error {
	//设置租约时间
	resp, err := s.cli.Grant(context.Background(), lease)
	if err != nil {
		return err
	}
	//注册服务并绑定租约
	_, err = s.cli.Put(context.Background(), s.key, s.val, clientv3.WithLease(resp.ID))
	if err != nil {
		return err
	}
	//设置续租 定期发送需求请求
	leaseRespChan, err := s.cli.KeepAlive(context.Background(), resp.ID)

	if err != nil {
		return err
	}
	s.leaseID = resp.ID
	log.Println(s.leaseID)
	s.keepAliveChan = leaseRespChan
	log.Printf("Put key:%s  val:%s  success!", s.key, s.val)
	return nil
}

//ListenLeaseRespChan 监听 续租情况
func (s *ServiceRegister) ListenLeaseRespChan() {
	for leaseKeepResp := range s.keepAliveChan {
		log.Println("续约成功", leaseKeepResp)
	}
	log.Println("关闭续租")
}

// Close 注销服务
func (s *ServiceRegister) Close() error {
	//撤销租约
	if _, err := s.cli.Revoke(context.Background(), s.leaseID); err != nil {
		return err
	}
	log.Println("撤销租约")
	return s.cli.Close()
}

func main() {
	var endpoints = []string{"localhost:2379"}
	ser, err := NewServiceRegister(endpoints, "/web/node1", "localhost:8000", 5)
	if err != nil {
		log.Fatalln(err)
	}
	//监听续租相应chan
	go ser.ListenLeaseRespChan()
	select {
	// case <-time.After(20 * time.Second):
	// 	ser.Close()
	}
}
```

主动退出服务时，可以调用Close()方法，撤销租约，从而注销服务。

### 服务发现

根据etcd的`v3 API`，很容易想到使用`Watch`监视某类服务，通过`Watch`感知服务的`添加`，`修改`或`删除`操作，修改服务列表。

```go
package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/coreos/etcd/mvcc/mvccpb"
	"go.etcd.io/etcd/clientv3"
)

//ServiceDiscovery 服务发现
type ServiceDiscovery struct {
	cli        *clientv3.Client  //etcd client
	serverList map[string]string //服务列表
	lock       sync.Mutex
}

//NewServiceDiscovery  新建发现服务
func NewServiceDiscovery(endpoints []string) *ServiceDiscovery {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}

	return &ServiceDiscovery{
		cli:        cli,
		serverList: make(map[string]string),
	}
}

//WatchService 初始化服务列表和监视
func (s *ServiceDiscovery) WatchService(prefix string) error {
	//根据前缀获取现有的key
	resp, err := s.cli.Get(context.Background(), prefix, clientv3.WithPrefix())
	if err != nil {
		return err
	}

	for _, ev := range resp.Kvs {
		s.SetServiceList(string(ev.Key), string(ev.Value))
	}

	//监视前缀，修改变更的server
	go s.watcher(prefix)
	return nil
}

//watcher 监听前缀
func (s *ServiceDiscovery) watcher(prefix string) {
	rch := s.cli.Watch(context.Background(), prefix, clientv3.WithPrefix())
	log.Printf("watching prefix:%s now...", prefix)
	for wresp := range rch {
		for _, ev := range wresp.Events {
			switch ev.Type {
			case mvccpb.PUT: //修改或者新增
				s.SetServiceList(string(ev.Kv.Key), string(ev.Kv.Value))
			case mvccpb.DELETE: //删除
				s.DelServiceList(string(ev.Kv.Key))
			}
		}
	}
}

//SetServiceList 新增服务地址
func (s *ServiceDiscovery) SetServiceList(key, val string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.serverList[key] = string(val)
	log.Println("put key :", key, "val:", val)
}

//DelServiceList 删除服务地址
func (s *ServiceDiscovery) DelServiceList(key string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.serverList, key)
	log.Println("del key:", key)
}

//GetServices 获取服务地址
func (s *ServiceDiscovery) GetServices() []string {
	s.lock.Lock()
	defer s.lock.Unlock()
	addrs := make([]string, 0)

	for _, v := range s.serverList {
		addrs = append(addrs, v)
	}
	return addrs
}

//Close 关闭服务
func (s *ServiceDiscovery) Close() error {
	return s.cli.Close()
}

func main() {
	var endpoints = []string{"localhost:2379"}
	ser := NewServiceDiscovery(endpoints)
	defer ser.Close()
	ser.WatchService("/web/")
	ser.WatchService("/gRPC/")
	for {
		select {
		case <-time.Tick(10 * time.Second):
			log.Println(ser.GetServices())
		}
	}
}
```
运行：
```
#运行服务发现
$go run discovery.go
watching prefix:/web/ now...
put key : /web/node1 val:localhost:8000
[localhost:8000]

#另一个终端运行服务注册
$go run register.go
Put key:/web/node1 val:localhost:8000 success!
续约成功 cluster_id:14841639068965178418 member_id:10276657743932975437 revision:29 raft_term:7 
续约成功 cluster_id:14841639068965178418 member_id:10276657743932975437 revision:29 raft_term:7 
...
```

### 总结
基于 Raft 算法的 etcd 天生是一个强一致性高可用的服务存储目录，用户可以在 etcd 中注册服务，并且对注册的服务设置key TTL，定时保持服务的心跳以达到监控健康状态的效果。通过在 etcd 指定的主题下注册的服务也能在对应的主题下查找到。

为了确保连接，我们可以在每个服务机器上都部署一个 Proxy 模式的 etcd，这样就可以确保能访问 etcd 集群的服务都能互相连接。

参考：
* https://segmentfault.com/a/1190000020944777
* https://blog.csdn.net/blogsun/article/details/102861648
* https://www.infoq.cn/article/etcd-interpretation-application-scenario-implement-principle/
