# nsq-v0.1.1README
### NSQ

An infrastructure component designed to support highly available, distributed, fault tolerant, "guaranteed" message delivery.    
一个支持高可用，分布式，容错性以及可靠消息传递的消息队列    

#### Background

`simplequeue` was developed as a dead-simple in-memory message queue. It spoke HTTP and had no knowledge (or care) for the data you put in or took out. Life was good.    
simplequeue是一个十分简单的内存消息队列（不做持久化）。他基于HTTP并且对内部传递的消息无感知，简洁而美秒。    

We used `simplequeue` as the foundation for a distributed message queue. In production, we silo'd a `simplequeue` right where messages were produced (ie. frontends) and effectively reduced the potential for data loss in a system which did not persist messages (by guaranteeing that the loss of any single `simplequeue` would not prevent the rest of the message producers or workers, to function).    
我们使用simplequeue来实现消息分发。simplequeue对接生产端，并且不对数据进行持久化来减少消息丢失的风险。    

We added `pubsub`, an HTTP server to aggregate streams and provide a long-lived `/sub` endpoint. We leveraged `pubsub` to transmit streams across data-centers in order to daisy chain a given feed to various downstream services. A nice property of this setup is that producers are de-coupled from downstream consumers (a downstream consumer only needs to know of the `pubsub` to receive data).    
pubsub方式。使用一个HTTP服务来聚合各个流并且对外提供一个长久有效地可订阅端。为了保证在总线型拓扑结构上将数据传递到多个下游服务，使用发布订阅模式在各个数据节点之间发送数据流。    

There are a few issues with this combination of tools...    
使用上述组合实现上还有一些问题需要解决    

One is simply the operational complexity of having to setup the data pipe to begin with. Often this involves services setup as follows:    
其中一个问题是设置数据管道十分复杂。通常相关的服务的设置规则如下：    
```c
    `api` > `simplequeue` > `queuereader` > `pubsub` > `ps_to_http` > `simplequeue` > `queuereader`
```

Of particular note are the `pubsub` > `ps_to_http` links. We repeatedly encounter the problem of consuming a single stream with the desire to avoid a SPOF. You have 2 options, none ideal. Often we just put the `ps_to_http` process on a single box and pray. Alternatively we've chosen to consume the full stream multiple times but only process a % of the stream on a given host (essentially sharding). To make things even more complicated we need to repeat this chain for each stream of data we're interested in.    
值得注意的一点是，从pubsub到ps_to_http的过程。在该关键过程中我们十分希望避免单点。另外我们通常消费所有流数据但是只处理其中一部分。同时我们在每个链路上都重复这一过程。    

Messages traveling through the system have no guarantee that they will be delivered to a client and the responsibility of requeueing is placed on the client. This churn of messages being passed back and forth increases the potential for errors resulting in message loss.    
消息的传递时不保证顺序性的，需要客户端自己做排序。这种传递方式也会导致错误发生。    

#### Enter NSQ

`NSQ` is designed to address the fragile nature of the combination of components listed above as well as provide high-availability as a byproduct of a messaging pattern that includes no SPOF. It also addresses the need for stronger guarantees around the delivery of a message.    
NSQ就是为了解决上述问题并且提供高可用性并消灭SPOF（单点），同时也实现了消息的可靠传递。    

A single `nsqd` process handles multiple "topics" (by convention, this would previously have been referred to as a "stream"). Second, a topic can have multiple "channels". In practice, a channel maps to a downstream service. Each channel receives all the messages from a topic. The channels buffer data independently of each other, preventing a slow consumer from causing a backlog for other channels. A channel can have multiple clients, a message (assuming successful delivery) will only be delivered to one of the connected clients, at random.    
每一个nsqd进程会承载多个topic（也就是上文说的流），同时每个topic会有多个channels，实际上一个channel对应一个下游服务。每个channel都会传递该topic所有的消息，且不同的channel之间的数据是互相独立的以此来防止出现一个慢速消费者导致的消息堆积问题。一个channel可以有多个客户端连接，一条消息只能随机的被一个客户端成功消费。    

For example, the "decodes" topic could have a channel for "clickatron", "spam", and "fishnet", etc. The benefit should be easy to see, there are no additional services needed to be setup for new queues or to daisy chain a new downstream service.    
例如我们有docodes的topic，其中有多个channels。这样的好处是当有新的队列或者新增新的下游服务时，我们的服务不需要额外的增加。    

`NSQ` is fully distributed with no single broker or point of failure. `nsqd` clients (aka "queuereaders") are connected over TCP sockets to **all** `nsqd` instances providing the specified topic. There are no middle-men, no brokers, and no SPOF. The *topology* solves the problems described above:    
NSQ是分布式的来解决单点问题。nsqd client通过tcp与其他所有同一个topic的nsqd连接。这其中并没有中间件，也没有broker。这种拓扑结构可以解决上面说的问题，即SPOF。    
```
    NSQ     NSQ    NSQ
      \     /\     /
       \   /  \   /
        \ /    \ /
         X      X
        / \    / \
       /   \  /   \
    ...  consumers  ...
```

You don't have to deal with figuring out how to robustly distribute an aggregated feed, instead you consume directly from *all* producers. It also doesn't *technically* matter which client connects to which `NSQ`, as long as there are enough clients connected to all producers to satisfy the volume of messages, you're guaranteed that all will be delivered downstream.    
你不用去关心你需要对接哪个聚合流，你直接消费所有的生产者。同时你也不用关心client和NSQ的对应关系。只要有足够多的client参与消费，就能保证消息全部被传输。    

It's also worth pointing out the bandwidth efficiency when you have >1 consumers of a topic. You're no longer daisy chaining multiple copies of the stream over the network, it's all happening right at the source. Moving away from HTTP as the only available transport also significantly reduces the per-message overhead.    
值得一提的是，当你拥有多个消费者的时候，你就不需要在重新部署一套链路系统了，直接在源头就能解决这个问题。同时使用HTTP之外的协议保证了消息传递的高效。    

#### Message Delivery Guarantees

`NSQ` guarantees that a message will be delivered **at least once**. Duplicate messages are possible and downstream systems should be designed to perform idempotent operations.    
NSQ保证所有的消息至少能被投递一次。消息多次投递是可能发生的，所以消费方需要保证消费的幂等性。    

It accomplishes this by performing a handshake with the client, as follows:    
该机制通过与client的握手机制实现：    

  1. client GET message
  2. `NSQ` sends message and stores in temporary internal location
  3. client replies SUCCESS or FAIL
     * if client does not reply `NSQ` will automatically timeout and requeue the message
  4. `NSQ` requeues on FAIL and purges on SUCCESS

1. 客户端等待获取消息
2. NSQ收到消息并且存储在临时内存中
3. client恢复成功或者失败
    * 超时未收到恢复，NSQ会重新将消息载入队列
4. NSQ当收到成功消息则删除消息，失败消息则会将消息重新入队

#### Lookup Service (nsqlookupd)

`NSQ` includes a helper application, `nsqlookupd`, which provides a directory service where queuereaders can lookup the addresses of `NSQ` instances that contain the topics they are interested in subscribing to. This decouples the consumers from the producers (they both individually only need to have intimate knowledge of `nsqlookupd`, never each other).    
NSQ还存在一个辅助应用nsqlookup，他可以提供一个目录给nsqd client来查找他们希望订阅的topic。以此来讲生产者和消费者进行了解耦。    

At a lower level each `nsqd` has a long-lived connection to `nsqlookupd` over which it periodically pushes it's state. This data is used to inform which addresses `nsqlookupd` will give to queuereaders. The heuristic could be based on depth, number of connected queuereaders or naive strategies like round-robin, etc. The goal is to ensure that all producers are being read from.  On the client side an HTTP interface is exposed for queuereaders to poll.    
在下层，每个nsqd进程都会定期的向nsqlookup进程发送自己的状态。这些信息中可以用来决策nsqdloockup进程向nsqd client提供哪些地址，这个策略可以是基于深度，client的个数或者简单的循环等简单的策略。目的是为了确保所有的生产者都能被消费到。同时也会暴露给client一个http接口来选取地址。    

High availability of `nsqlookupd` is achieved by running multiple instances. They don't communicate directly to each other and don't require strong data consistency between themselves. The data is considered *eventually* consistent, the queuereaders randomly choose a `nsqlookupd` to poll. Stale (or otherwise inaccessible) nodes don't grind the system to a halt.    
多实例保证了nsqlookup进程的高可用。他们之间并不会直接进行通信，并且不要求强一致性。只要保证最终一致性，client随机的选取一个其中一个连接。过期节点和闭塞节点并不能导致系统的不可用。        
