### etcd简介
[etcd](https://github.com/etcd-io/etcd)是开源的、高可用的分布式key-value存储系统，可用于配置共享和服务的注册和发现，它专注于：

* 简单：定义清晰、面向用户的API（gRPC）

* 安全：可选的客户端TLS证书自动认证

* 快速：支持每秒10,000次写入

* 可靠：基于Raft算法确保强一致性

##### etcd与redis差异
etcd和redis都支持键值存储，也支持分布式特性，redis支持的数据格式更加丰富，但是他们两个定位和应用场景不一样，关键差异如下：

* redis在分布式环境下不是强一致性的，可能会丢失数据，或者读取不到最新数据

* redis的数据变化监听机制没有etcd完善

* etcd强一致性保证数据可靠性，导致性能上要低于redis

* etcd和ZooKeeper是定位类似的项目，跟redis定位不一样

##### 为什么用 etcd 而不用ZooKeeper？
相较之下，ZooKeeper有如下缺点：

* `复杂`：ZooKeeper的部署维护复杂，管理员需要掌握一系列的知识和技能；而 Paxos 强一致性算法也是素来以复杂难懂而闻名于世；另外，ZooKeeper的使用也比较复杂，需要安装客户端，官方只提供了 Java 和 C 两种语言的接口。

* `难以维护`：Java 编写。这里不是对 Java 有偏见，而是 Java 本身就偏向于重型应用，它会引入大量的依赖。而运维人员则普遍希望保持强一致、高可用的机器集群尽可能简单，维护起来也不易出错。

* `发展缓慢`：Apache 基金会项目特有的“Apache Way”在开源界饱受争议，其中一大原因就是由于基金会庞大的结构以及松散的管理导致项目发展缓慢。

而 etcd 作为一个后起之秀，其优点也很明显。

* `简单`：使用 Go 语言编写部署简单；使用 HTTP 作为接口使用简单；使用 Raft 算法保证强一致性让用户易于理解。

* `数据持久化`：tcd 默认数据一更新就进行持久化。

* `安全`：etcd 支持 SSL 客户端安全认证。

### 单机部署

（1）到etcd的github地址，下载最新的安装包（目前最新版本：v3.4.7）

下载地址：https://github.com/etcd-io/etcd/releases/

（2）解压，把`etcd`和`etcdctl`文件复制到已经配置了环境变量的目录中

* 方法一：把`etcd`和`etcdctl`文件复制到`GOBIN`目录下。

* 方法二：在环境变量里添加`etcd`和`etcdctl`文件所在的目录。

（3）验证是否安装成功
```
$ etcd --version
etcd Version: 3.4.7
Git SHA: e694b7bb0
Go Version: go1.12.17
Go OS/Arch: linux/amd64
```

正常显示etcd版本信息，则证明安装成功。

### API学习
`etcdctl`用于与`etcd`交互的控制台程序。`API`版本可以通过`ETCDCTL_API`环境变量设置为2或3版本。默认情况下，`v3.4`以上的`etcdctl`使用`v3 API`，`v3.3`及更早的版本默认使用`v2 API`。

> 注意：用`v2 API`创建的任何key将不能通过`v3 API`查询。同样，用`v3 API`创建的任何key将不能通过`v2 API`查询。

运行etcd，在终端输入：`etcd`

在另一个终端运行ctcdctl测试。
```
#查看默认API版本
$ etcdctl version  
etcdctl version: 3.4.7
API version: 3.4  #v3 API

#写入key:/test/foo value:hello etcd (双引号可去掉)
$ etcdctl put /test/foo "hello etcd" 
OK
$ etcdctl get /test/foo
/test/foo
hello etcd

#手动切换到v2 API
$ export ETCDCTL_API=2  
$ etcdctl --version
etcdctl version: 3.4.7
API version: 2

$ etcdctl get /test/foo
Error:  client: response is invalid json. The endpoint is probably not valid etcd cluster endpoint      #查询不到/test/foo的值
```

#### 写入key
```
$ etcdctl put foo bar
OK
```
#### 读取key值
```
$ etcdctl get foo
foo
bar

#只是获取值
$ etcdctl get foo --print-value-only
bar
```

```
$ etcdctl put foo1 bar1
$ etcdctl put foo2 bar2
$ etcdctl put foo3 bar3

#获取从foo到foo3的值，不包括foo3
$ etcdctl get foo foo3 --print-value-only 
bar
bar1
bar2

# 获取前缀为foo的值
$ etcdctl get --prefix foo --print-value-only
bar
bar1
bar2
bar3

#获取符合前缀的前两个值
$ etcdctl get --prefix --limit=2 foo --print-value-only
bar
bar1
```

#### 删除key
```
#删除foo
$ etcdctl del foo
1

#删除foo到foo2，不包括foo2
$ etcdctl del foo foo2
1
#删除key前缀为foo的
$ etcdctl del --prefix foo
2
```

#### 监视值变化
```
#监视foo单个key
$ etcdctl watch foo
#另一个控制台执行： etcdctl put foo bar
PUT
foo
bar

#同时监视多个值
$ etcdctl watch -i
$ watch foo
$ watch zoo
# 另一个控制台执行: etcdctl put foo bar
PUT
foo
bar
# 另一个控制台执行: etcdctl put zoo val
PUT
zoo
val

#监视foo前缀的key
$ etcdctl watch --prefix foo
#另一个控制台执行： etcdctl put foo1 bar1
PUT
foo1
bar1
#另一个控制台执行： etcdctl put fooz1 barz1
PUT
fooz1
barz1
```

#### 设置租约（Grant leases）

当一个key被绑定到一个租约上时，它的生命周期与租约的生命周期绑定。

```
#设置60秒后过期时间
$ etcdctl lease grant 60
lease 32695410dcc0ca06 granted with TTL(60s)

#把foo和租约绑定，设置成60秒后过期
$ etcdctl put --lease=32695410dcc0ca06 foo bar
OK
$ etcdctl get foo
foo
bar

#60秒后，获取不到foo
$ etcdctl get foo
#返回空
```

#### 主动撤销租约（Revoke leases）

通过租赁ID（此处指：`32695410dcc0ca06`）撤销租约。`撤销租约将删除其所有绑定的key`。

```
$ etcdctl lease grant 60
lease 32695410dcc0ca06 granted with TTL(60s)
$ etcdctl put foo bar --lease=32695410dcc0ca06
OK

#主动撤销租约
$ etcdctl lease revoke 32695410dcc0ca06
lease 32695410dcc0ca06 revoked

$ etcdctl get foo
#返回空
```

#### 续租约（Keep leases alive）

通过刷新其TTL来保持租约的有效，使其不会过期。

```
#设置60秒后过期租约
$ etcdctl lease grant 60
lease 32695410dcc0ca06 granted with TTL(60s)

#把foo和租约绑定，设置成60秒后过期
$ etcdctl put foo bar --lease=32695410dcc0ca06

#续租约，自动定时执行续租约，续约成功后每次租约为60秒
$ etcdctl lease keep-alive 32695410dcc0ca06
lease 32695410dcc0ca06 keepalived with TTL(60)
lease 32695410dcc0ca06 keepalived with TTL(60)
lease 32695410dcc0ca06 keepalived with TTL(60)
...
```

#### 获取租约信息（Get lease information）

 获取租约信息，以便续租或查看租约是否仍然存在或已过期
 
```
#设置500秒TTL
$ etcdctl lease grant 500
lease 694d5765fc71500b granted with TTL(500s)

#keyzoo1绑定694d5765fc71500b租约
$ etcdctl put zoo1 val1 --lease=694d5765fc71500b
OK

#查看租约信息，remaining(132s)剩余有效时间132秒；--keys获取租约绑定的key
$ etcdctl lease timetolive --keys 694d5765fc71500b
lease 694d5765fc71500b granted with TTL(500s), remaining(132s), attached keys([zoo1])
```

值得注意的地方，一个租约可以绑定多个`key`
```
$ etcdctl lease grant 500
lease 694d5765fc71500b granted with TTL(500s)

$ etcdctl put zoo1 val1 --lease=694d5765fc71500b
OK

$ etcdctl put zoo2 val2 --lease=694d5765fc71500b
OK
```
当租约过期后，所有key值会被删除。

当一个租约只绑定了一个`key`时，想删除这个`key`，最好的办法是撤销它的租约，而不是直接删除这个`key`。

看下面这个例子：
```
#方法一：直接删除`key`
#设置租约并绑定zoo1
$ etcdctl lease grant 60
lease 694d71f80ed8bf1e granted with TTL(60s)
$ etcdctl put zoo1 val1 --lease=694d71f80ed8bf1e
OK

#续租约
$ etcdctl lease keep-alive 694d71f80ed8bf1e
lease 694d71f80ed8bf1e keepalived with TTL(60)

#另一个控制台执行：etcdctl del zoo1

#单纯删除key后，续约操作还会一直进行，造成内存泄露
lease 694d71f80ed8bf1e keepalived with TTL(60)
lease 694d71f80ed8bf1e keepalived with TTL(60)
lease 694d71f80ed8bf1e keepalived with TTL(60)
...
```

```
方法二：撤销`key`的租约
#设置租约并绑定zoo1
$ etcdctl lease grant 60
lease 694d71f80ed8bf1e granted with TTL(60s)
$ etcdctl put zoo1 val1 --lease=694d71f80ed8bf1e
OK

#续租约
$ etcdctl lease keep-alive 694d71f80ed8bf1e
lease 694d71f80ed8bf1e keepalived with TTL(60)
lease 694d71f80ed8bf1e keepalived with TTL(60)

#另一个控制台执行：etcdctl lease revoke 694d71f80ed8bf1e

#续约操作并退出
lease 694d71f80ed8bf1e expired or revoked.
```

当租约没有绑定`key`时，应主动把它撤销掉。

### 应用场景

根据以上特性和API，etcd有应用场景以下应用场景：

#### 场景一：服务发现
服务发现要解决的也是分布式系统中最常见的问题之一，即在同一个分布式集群中的进程或服务，要如何才能找到对方并建立连接。本质上来说，服务发现就是想要了解集群中是否有进程在监听 udp 或 tcp 端口，并且通过名字就可以查找和连接。

#### 场景二：配置中心
etcd的应用场景优化都是围绕存储的东西是“配置” 来设定的。
* 配置的数据量通常都不大，所以默认etcd的存储上限是1GB
* 配置通常对历史版本信息是比较关心的，所以etcd会保存 版本（revision） 信息
* 配置变更是比较常见的，并且业务程序会需要实时知道，所以etcd提供了watch机制，基本就是实时通知配置变化
* 配置的准确性一致性极其重要，所以etcd采用raft算法，保证系统的CP
* 同一份配置通常会被大量客户端同时访问，针对这个做了grpc proxy对同一个key的watcher做了优化
* 配置会被不同的业务部门使用，提供了权限控制和namespace机制

#### 场景三：负载均衡
此处指的负载均衡均为软负载均衡，分布式系统中，为了保证服务的高可用以及数据的一致性，通常都会把数据和服务部署多份，以此达到对等服务，即使其中的某一个服务失效了，也不影响使用。由此带来的坏处是数据写入性能下降，而好处则是数据访问时的负载均衡。因为每个对等服务节点上都存有完整的数据，所以用户的访问流量就可以分流到不同的机器上。

#### 场景四：分布式锁
因为 etcd 使用 Raft 算法保持了数据的强一致性，某次操作存储到集群中的值必然是全局一致的，所以很容易实现分布式锁。

#### 场景五：集群监控与 Leader 竞选
通过 etcd 来进行监控实现起来非常简单并且实时性强。

* 前面几个场景已经提到 Watcher 机制，当某个节点消失或有变动时，Watcher 会第一时间发现并告知用户。
* 节点可以设置TTL key，比如每隔 30s 发送一次心跳使代表该机器存活的节点继续存在，否则节点消失。

这样就可以第一时间检测到各节点的健康状态，以完成集群的监控要求。

另外，使用分布式锁，可以完成 Leader 竞选。这种场景通常是一些长时间 CPU 计算或者使用 IO 操作的机器，只需要竞选出的 Leader 计算或处理一次，就可以把结果复制给其他的 Follower。从而避免重复劳动，节省计算资源。

参考：
* https://github.com/etcd-io/etcd
* https://etcd.io/docs/v3.4.0/dev-guide/interacting_v3/
* https://www.infoq.cn/article/etcd-interpretation-application-scenario-implement-principle/