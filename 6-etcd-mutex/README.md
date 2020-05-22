### etcd分布式锁及事务

### 前言

`分布式锁`是控制分布式系统之间同步访问共享资源的一种方式。在分布式系统中，常常需要协调他们的动作。如果不同的系统或是同一个系统的不同主机之间共享了一个或一组资源，那么访问这些资源的时候，往往需要互斥来防止彼此干扰来保证一致性，在这种情况下，便需要使用到分布式锁。

### etcd分布式锁设计

1. `排他性`：任意时刻，只能有一个机器的一个线程能获取到锁。

通过在etcd中存入key值来实现上锁，删除key实现解锁，参考下面伪代码：

```go
func Lock(key string, cli *clientv3.Client) error {
    //获取key，判断是否存在锁
	resp, err := cli.Get(context.Background(), key)
	if err != nil {
		return err
	}
	//锁存在，返回上锁失败
	if len(resp.Kvs) > 0 {
		return errors.New("lock fail")
	}
	_, err = cli.Put(context.Background(), key, "lock")
	if err != nil {
		return err
	}
	return nil
}
//删除key，解锁
func UnLock(key string, cli *clientv3.Client) error {
	_, err := cli.Delete(context.Background(), key)
	return err
}
```

当发现已上锁时，直接返回lock fail。也可以处理成等待解锁，解锁后竞争锁。
```go
//等待key删除后再竞争锁
func waitDelete(key string, cli *clientv3.Client) {
	rch := cli.Watch(context.Background(), key)
	for wresp := range rch {
		for _, ev := range wresp.Events {
			switch ev.Type {
			case mvccpb.DELETE: //删除
				return
			}
		}
	}
}
```

2. `容错性`：只要分布式锁服务集群节点大部分存活，client就可以进行加锁解锁操作。
`etcd`基于`Raft`算法，确保集群中数据一致性。

3. `避免死锁`：分布式锁一定能得到释放，即使client在释放之前崩溃。
上面分布式锁设计有缺陷，假如client获取到锁后程序直接崩了，没有解锁，那其他线程也无法拿到锁，导致死锁出现。
通过给key设定`leases`来避免死锁，但是`leases`过期时间设多长呢？假如设了30秒，而上锁后的操作比30秒大，会导致以下问题：

* 操作没完成，锁被别人占用了，不安全

* 操作完成后，进行解锁，这时候把别人占用的锁解开了

`解决方案`：给key添加过期时间后，以`Keep leases alive`方式延续`leases`，当client正常持有锁时，锁不会过期；当client程序崩掉后，程序不能执行`Keep leases alive`，从而让锁过期，避免死锁。看以下伪代码：

```go
//上锁
func Lock(key string, cli *clientv3.Client) error {
    //获取key，判断是否存在锁
	resp, err := cli.Get(context.Background(), key)
	if err != nil {
		return err
	}
	//锁存在，等待解锁后再竞争锁
	if len(resp.Kvs) > 0 {
		waitDelete(key, cli)
		return Lock(key)
	}
    //设置key过期时间
	resp, err := cli.Grant(context.TODO(), 30)
	if err != nil {
		return err
	}
	//设置key并绑定过期时间
	_, err = cli.Put(context.Background(), key, "lock", clientv3.WithLease(resp.ID))
	if err != nil {
		return err
	}
	//延续key的过期时间
	_, err = cli.KeepAlive(context.TODO(), resp.ID)
	if err != nil {
		return err
	}
	return nil
}
//通过让key值过期来解锁
func UnLock(resp *clientv3.LeaseGrantResponse, cli *clientv3.Client) error {
	_, err := cli.Revoke(context.TODO(), resp.ID)
	return err
}
```

经过以上步骤，我们初步完成了分布式锁设计。其实官方已经实现了分布式锁，它大致原理和上述有出入，接下来我们看下如何使用官方的分布式锁。

### etcd分布式锁使用

```go
func ExampleMutex_Lock() {
	cli, err := clientv3.New(clientv3.Config{Endpoints: endpoints})
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	// create two separate sessions for lock competition
	s1, err := concurrency.NewSession(cli)
	if err != nil {
		log.Fatal(err)
	}
	defer s1.Close()
	m1 := concurrency.NewMutex(s1, "/my-lock/")

	s2, err := concurrency.NewSession(cli)
	if err != nil {
		log.Fatal(err)
	}
	defer s2.Close()
	m2 := concurrency.NewMutex(s2, "/my-lock/")

	// acquire lock for s1
	if err := m1.Lock(context.TODO()); err != nil {
		log.Fatal(err)
	}
	fmt.Println("acquired lock for s1")

	m2Locked := make(chan struct{})
	go func() {
		defer close(m2Locked)
		// wait until s1 is locks /my-lock/
		if err := m2.Lock(context.TODO()); err != nil {
			log.Fatal(err)
		}
	}()

	if err := m1.Unlock(context.TODO()); err != nil {
		log.Fatal(err)
	}
	fmt.Println("released lock for s1")

	<-m2Locked
	fmt.Println("acquired lock for s2")

	// Output:
	// acquired lock for s1
	// released lock for s1
	// acquired lock for s2
}
```
此代码来源于[官方文档](https://github.com/etcd-io/etcd/blob/master/clientv3/concurrency/example_mutex_test.go)，etcd分布式锁使用起来很方便。

### etcd事务

顺便介绍一下etcd事务，先看这段伪代码：

```go
Txn(context.TODO()).If(//如果以下判断条件成立
	Compare(Value(k1), "<", v1),
	Compare(Version(k1), "=", 2)
).Then(//则执行Then代码段
	OpPut(k2,v2), OpPut(k3,v3)
).Else(//否则执行Else代码段
	OpPut(k4,v4), OpPut(k5,v5)
).Commit()//最后提交事务
```

使用例子，代码来自[官方文档](https://github.com/etcd-io/etcd/blob/master/clientv3/example_kv_test.go)：

```go
func ExampleKV_txn() {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: dialTimeout,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	kvc := clientv3.NewKV(cli)

	_, err = kvc.Put(context.TODO(), "key", "xyz")
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	_, err = kvc.Txn(ctx).
		// txn value comparisons are lexical
		If(clientv3.Compare(clientv3.Value("key"), ">", "abc")).
		// the "Then" runs, since "xyz" > "abc"
		Then(clientv3.OpPut("key", "XYZ")).
		// the "Else" does not run
		Else(clientv3.OpPut("key", "ABC")).
		Commit()
	cancel()
	if err != nil {
		log.Fatal(err)
	}

	gresp, err := kvc.Get(context.TODO(), "key")
	cancel()
	if err != nil {
		log.Fatal(err)
	}
	for _, ev := range gresp.Kvs {
		fmt.Printf("%s : %s\n", ev.Key, ev.Value)
	}
	// Output: key : XYZ
}
```

### 总结

如果发展到分布式服务阶段，且对数据的可靠性要求很高，选`etcd`实现分布式锁不会错。介于对`ZooKeeper`好感度不强，这里就不介绍`ZooKeeper`分布式锁了。一般的`Redis`分布式锁，可能出现锁丢失的情况（如果你是Java开发者，可以使用Redisson客户端实现分布式锁，据说不会出现锁丢失的情况）。