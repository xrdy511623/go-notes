
---
切⽚集群：数据增多了，是该加内存还是加实例？
---


# 1 纵向扩展的问题-RDB持久化时fork子进程会阻塞主线程

假设有这么⼀个需求：要⽤Redis保存5000万个键值对，每个键值对⼤约是512B，为了能快速部署并对
外提供服务，我们采⽤云主机来运⾏Redis实例，那么，该如何选择云主机的内存容量呢？

我们可以粗略地计算⼀下，这些键值对所占的内存空间⼤约是25GB（5000万*512B）。所以，我们可能想到的
第⼀个⽅案就是：选择⼀台32GB内存的云主机来部署Redis。因为32GB的内存能保存所有数据，⽽且还留
有7GB，可以保证系统的正常运⾏。同时，还采⽤RDB对数据做持久化，以确保Redis实例故障后，还能
从RDB恢复数据。

但是，在使⽤的过程中，我发现，Redis的响应有时会⾮常慢。后来，我们使⽤INFO命令查看Redis的
latest_fork_usec指标值（表⽰最近⼀次fork的耗时），结果显⽰这个指标值特别⾼，快到秒级别了。
这跟Redis的持久化机制有关系。在使⽤RDB进⾏持久化时，Redis会fork⼦进程来完成，fork操作的⽤时和
Redis的数据量是正相关的，⽽fork在执⾏时会阻塞主线程。数据量越⼤，fork操作造成的主线程阻塞的时
间越⻓。所以，在使⽤RDB对25GB的数据进⾏持久化时，数据量较⼤，后台运⾏的⼦进程在fork创建时阻
塞了主线程，于是就导致Redis响应变慢了。

看来，第⼀个⽅案显然是不可⾏的，我们必须要寻找其他的⽅案。这个时候，我们注意到了Redis的切⽚集
群。虽然组建切⽚集群⽐较⿇烦，但是它可以保存⼤量数据，⽽且对Redis主线程的阻塞影响较⼩。

切⽚集群，也叫分⽚集群，就是指启动多个Redis实例组成⼀个集群，然后按照⼀定的规则，把收到的数据
划分成多份，每⼀份⽤⼀个实例来保存。回到我们刚刚的场景中，如果把25GB的数据平均分成5份（当然，
也可以不做均分），使⽤5个实例来保存，每个实例只需要保存5GB数据。如下图所⽰：





![cluster-store-data-demo.png](images%2Fcluster-store-data-demo.png)





那么，在切⽚集群中，实例在为5GB数据⽣成RDB时，数据量就⼩了很多，fork⼦进程⼀般不会给主线程带
来较⻓时间的阻塞。采⽤多个实例保存数据切⽚后，我们既能保存25GB数据，⼜避免了fork⼦进程阻塞主
线程⽽导致的响应突然变慢。

在实际应⽤Redis时，随着⽤⼾或业务规模的扩展，保存⼤量数据的情况通常是⽆法避免的。⽽切⽚集群，
就是⼀个⾮常好的解决⽅案。


# 2 如何保存更多数据？

在刚刚的案例⾥，为了保存⼤量数据，我们使⽤了⼤内存云主机和切⽚集群两种⽅法。实际上，这两种⽅法
分别对应着Redis应对数据量增多的两种⽅案：纵向扩展（scale up）和横向扩展（scale out）。

纵向扩展：升级单个Redis实例的资源配置，包括增加内存容量、增加磁盘容量、使⽤更⾼配置的CPU。
就像下图中，原来的实例内存是8GB，硬盘是50GB，纵向扩展后，内存增加到24GB，磁盘增加到
150GB。

横向扩展：横向增加当前Redis实例的个数，比如原来使⽤1个8GB内存、50GB磁盘的实例，现
在使⽤三个相同配置的实例。

那么，这两种⽅式的优缺点分别是什么呢？

⾸先，纵向扩展的好处是，实施起来简单、直接。不过，这个⽅案也⾯临两个潜在的问题。

第⼀个问题是，当使⽤RDB对数据进⾏持久化时，如果数据量增加，需要的内存也会增加，主线程fork⼦进
程时就可能会阻塞（⽐如刚刚的例⼦中的情况）。不过，如果你不要求持久化保存Redis数据，那么，纵向
扩展会是⼀个不错的选择。

不过，这时，你还要⾯对第⼆个问题：纵向扩展会受到硬件和成本的限制。这很容易理解，毕竟，把内存从
32GB扩展到64GB还算容易，但是，要想扩充到1TB，就会⾯临硬件容量和成本上的限制了。

与纵向扩展相⽐，横向扩展是⼀个扩展性更好的⽅案。这是因为，要想保存更多的数据，采⽤这种⽅案的
话，只⽤增加Redis的实例个数就⾏了，不⽤担⼼单个实例的硬件和成本限制。在⾯向百万、千万级别的⽤
⼾规模时，横向扩展的Redis切⽚集群会是⼀个⾮常好的选择。

不过，在只使⽤单个实例的时候，数据存在哪⼉，客⼾端访问哪⼉，都是⾮常明确的，但是，切⽚集群不可
避免地涉及到多个实例的分布式管理问题。要想把切⽚集群⽤起来，我们就需要解决两⼤问题：

数据切⽚后，在多个实例之间如何分布？
客⼾端怎么确定想要访问的数据在哪个实例上？

接下来，我们来看，redis cluster是如何解决这两个问题的。


# 3 数据切⽚和实例的对应分布关系

在切⽚集群中，数据需要分布在不同实例上，那么，数据和实例之间如何对应呢？这就和接下来我要讲的
Redis Cluster⽅案有关了。不过，我们要先弄明⽩切⽚集群和Redis Cluster的联系与区别。

实际上，切⽚集群是⼀种保存⼤量数据的通⽤机制，这个机制可以有不同的实现⽅案。在Redis 3.0之前，
官⽅并没有针对切⽚集群提供具体的⽅案。从3.0开始，官⽅提供了⼀个名为Redis Cluster的⽅案，⽤于实
现切⽚集群。Redis Cluster⽅案中就规定了数据和实例的对应规则。

具体来说，Redis Cluster⽅案采⽤哈希槽（Hash Slot，接下来我会直接称之为Slot），来处理数据和实例
之间的映射关系。在Redis Cluster⽅案中，⼀个切⽚集群共有16384个哈希槽，这些哈希槽类似于数据分
区，每个键值对都会根据它的key，被映射到⼀个哈希槽中。

具体的映射过程分为两⼤步：⾸先根据键值对的key，按照CRC16算法计算⼀个16 bit的值；然后，再⽤这
个16bit值对16384取模，得到0~16383范围内的模数，每个模数代表⼀个相应编号的哈希槽。

那么，这些哈希槽⼜是如何被映射到具体的Redis实例上的呢？

我们在部署Redis Cluster⽅案时，可以使⽤cluster create命令创建集群，此时，Redis会⾃动把这些槽平均
分布在集群实例上。例如，如果集群中有N个实例，那么，每个实例上的槽个数为16384/N个。

当然， 我们也可以使⽤cluster meet命令⼿动建⽴实例间的连接，形成集群，再使⽤cluster addslots命
令，指定每个实例上的哈希槽个数。

举个例⼦，假设集群中不同Redis实例的内存⼤⼩配置不⼀，如果把哈希槽均分在各个实例上，在保存相同
数量的键值对时，和内存⼤的实例相⽐，内存⼩的实例就会有更⼤的容量压⼒。遇到这种情况时，你可以根
据不同实例的资源配置情况，使⽤cluster addslots命令⼿动分配哈希槽。

下面这张图展示了，数据、哈希槽、实例这三者的映射分布情况。





![data-map-into-shard.png](images%2Fdata-map-into-shard.png)





⽰意图中的切⽚集群⼀共有3个实例，同时假设有5个哈希槽，我们⾸先可以通过下⾯的命令⼿动分配哈希
槽：实例1保存哈希槽0和1，实例2保存哈希槽2和3，实例3保存哈希槽4。

```shell
redis-cli -h 172.16.19.3 –p 6379 cluster addslots 0,1
redis-cli -h 172.16.19.4 –p 6379 cluster addslots 2,3
redis-cli -h 172.16.19.5 –p 6379 cluster addslots 4
```

在集群运⾏的过程中，key1和key2计算完CRC16值后，对哈希槽总个数5取模，再根据各⾃的模数结果，就
可以被映射到对应的实例1和实例3上了。

另外，需要注意的是，在⼿动分配哈希槽时，需要把16384个槽都分配完，否则Redis集群⽆法正常
⼯作。

好了，通过哈希槽，切⽚集群就实现了数据到哈希槽、哈希槽再到实例的分配。但是，即使实例有了哈希槽
的映射信息，客⼾端⼜是怎么知道要访问的数据在哪个实例上呢？


# 4 客户端如何定位数据？

在定位键值对数据时，它所处的哈希槽是可以通过计算得到的，这个计算可以在客⼾端发送请求时来执⾏。
但是，要进⼀步定位到实例，还需要知道哈希槽分布在哪个实例上。

⼀般来说，客⼾端和集群实例建⽴连接后，实例就会把哈希槽的分配信息发给客⼾端。但是，在集群刚刚创
建的时候，每个实例只知道⾃⼰被分配了哪些哈希槽，是不知道其他实例拥有的哈希槽信息的。

那么，客⼾端为什么可以在访问任何⼀个实例时，都能获得所有的哈希槽信息呢？这是因为，Redis实例会
把⾃⼰的哈希槽信息发给和它相连接的其它实例，来完成哈希槽分配信息的扩散。当实例之间相互连接后，
每个实例就有所有哈希槽的映射关系了。

客⼾端收到哈希槽信息后，会把哈希槽信息缓存在本地。当客⼾端请求键值对时，会先计算键所对应的哈希
槽，然后就可以给相应的实例发送请求了。

但是，在集群中，实例和哈希槽的对应关系并不是⼀成不变的，最常⻅的变化有两个：

在集群中，实例有新增或删除，Redis需要重新分配哈希槽；
为了负载均衡，Redis需要把哈希槽在所有实例上重新分布⼀遍。

此时，实例之间还可以通过相互传递消息，获得最新的哈希槽分配信息，但是，客⼾端是⽆法主动感知这些
变化的。这就会导致，它缓存的分配信息和最新的分配信息就不⼀致了，那该怎么办呢？

Redis Cluster⽅案提供了⼀种重定向机制，所谓的“重定向”，就是指，客⼾端给⼀个实例发送数据读写操
作时，这个实例上并没有相应的数据，客⼾端要再给⼀个新实例发送操作命令。

那客⼾端⼜是怎么知道重定向时的新实例的访问地址呢？当客⼾端把⼀个键值对的操作请求发给⼀个实例
时，如果这个实例上并没有这个键值对映射的哈希槽，那么，这个实例就会给客⼾端返回下⾯的MOVED命
令响应结果，这个结果中就包含了新实例的访问地址。

```shell
GET hello:key
(error) MOVED 13320 172.16.19.5:6379
```

其中，MOVED命令表⽰，客⼾端请求的键值对所在的哈希槽13320，实际是在172.16.19.5这个实例上。通
过返回的MOVED命令，就相当于把哈希槽所在的新实例的信息告诉给客⼾端了。这样⼀来，客⼾端就可以
直接和172.16.19.5连接，并发送操作请求了。

我画⼀张图来说明⼀下，MOVED重定向命令的使⽤⽅法。可以看到，由于负载均衡，Slot 2中的数据已经从
实例2迁移到了实例3，但是，客⼾端缓存仍然记录着“Slot 2在实例2”的信息，所以会给实例2发送命令。
实例2给客⼾端返回⼀条MOVED命令，把Slot 2的最新位置（也就是在实例3上），返回给客⼾端，客⼾端就
会再次向实例3发送请求，同时还会更新本地缓存，把Slot 2与实例的对应关系更新过来。






![cluster-move.png](images%2Fcluster-move.png)





需要注意的是，在上图中，当客⼾端给实例2发送命令时，Slot 2中的数据已经全部迁移到了实例3。在实际
应⽤时，如果Slot 2中的数据⽐较多，就可能会出现⼀种情况：客⼾端向实例2发送请求，但此时，Slot 2中
的数据只有⼀部分迁移到了实例3，还有部分数据没有迁移。在这种迁移部分完成的情况下，客⼾端就会收
到⼀条ASK报错信息，如下所⽰：

```shell
GET hello:key
(error) ASK 13320 172.16.19.5:6379
```

这个结果中的ASK命令就表⽰，客⼾端请求的键值对所在的哈希槽13320，在172.16.19.5这个实例上，但是
这个哈希槽正在迁移。此时，客⼾端需要先给172.16.19.5这个实例发送⼀个ASKING命令。这个命令的意思
是，让这个实例允许执⾏客⼾端接下来发送的命令。然后，客⼾端再向这个实例发送GET命令，以读取数据。

看起来好像有点复杂，我再借助图⽚来解释⼀下。

在下图中，Slot 2正在从实例2往实例3迁移，key1和key2已经迁移过去，key3和key4还在实例2。客⼾端向
实例2请求key2后，就会收到实例2返回的ASK命令。

ASK命令表⽰两层含义：第⼀，表明Slot数据还在迁移中；第⼆，ASK命令把客⼾端所请求数据的最新实例
地址返回给客⼾端，此时，客⼾端需要给实例3发送ASKING命令，然后再发送操作命令。






![cluster-ask.png](images%2Fcluster-ask.png)





和MOVED命令不同，ASK命令并不会更新客⼾端缓存的哈希槽分配信息。所以，在上图中，如果客⼾端再
次请求Slot 2中的数据，它还是会给实例2发送请求。这也就是说，ASK命令的作⽤只是让客⼾端能给新实例
发送⼀次请求，⽽不像MOVED命令那样，会更改本地缓存，让后续所有命令都发往新实例。


# 5 小结

本节主要讲了切⽚集群在保存⼤量数据⽅⾯的优势，以及基于哈希槽的数据分布机制和客⼾端定位键值对的⽅法。

在应对数据量扩容时，虽然增加内存这种纵向扩展的⽅法简单直接，但是会造成数据库的内存过⼤，导致性
能变慢。Redis切⽚集群提供了横向扩展的模式，也就是使⽤多个实例，并给每个实例配置⼀定数量的哈希
槽，数据可以通过键的哈希值映射到哈希槽，再通过哈希槽分散保存到不同的实例上。这样做的好处是扩展
性好，不管有多少数据，切⽚集群都能应对。

另外，集群的实例增减，或者是为了实现负载均衡⽽进⾏的数据重新分布，会导致哈希槽和实例的映射关系
发⽣变化，客⼾端发送请求时，会收到命令执⾏报错信息。了解了MOVED和ASK命令，你就不会为这类报
错⽽头疼了。

在Redis 3.0 之前，Redis官⽅并没有提供切⽚集群⽅案，但是，其实当时业界已经有了⼀些切⽚集群的⽅案，
例如基于客⼾端分区的ShardedJedis，基于代理的Codis、Twemproxy等。这些⽅案的应⽤早于Redis Cluster
⽅案，在⽀撑的集群实例规模、集群稳定性、客⼾端友好性⽅⾯也都有着各⾃的优势，下一节我们以codis方案为例，
讲讲其实现机制，以及实践经验。这样⼀来，当你再碰到业务发展带来的数据量巨⼤的难题时，就可以根据这些⽅案
的特点，选择合适的⽅案实现切⽚集群，以应对业务需求了。