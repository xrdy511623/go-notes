
---
codis切片集群
---

本节具体讲解 Codis 的关键技术实现原理，同时将 Codis 和 Redis
Cluster 进⾏对⽐，帮你选出最佳的集群⽅案。


# 1 Codis 的整体架构和基本流程

Codis 集群中包含了 4 类关键组件。

• codis server：这是进⾏了⼆次开发的 Redis 实例，其中增加了额
外的数据结构，⽀持数据迁移操作，主要负责处理具体的数据读写
请求。
• codis proxy：接收客户端请求，并把请求转发给 codis server。
• Zookeeper 集群：保存集群元数据，例如数据位置信息和 codis
proxy 信息。
• codis dashboard 和 codis fe：共同组成了集群管理⼯具。其中，
codis dashboard 负责执⾏集群管理⼯作，包括增删 codis
server、 codis proxy 和进⾏数据迁移。⽽ codis fe 负责提供
dashboard 的 Web 操作界⾯，便于我们直接在 Web 界⾯上进⾏
集群管理。

下面这张图展示了 Codis 集群的架构和关键组件。





![codis-structure.png](images%2Fcodis-structure.png)





下面具体讲讲 Codis 是如何处理请求的:

⾸先，为了让集群能接收并处理请求，我们要先使⽤ codis dashboard
设置 codis server 和 codis proxy 的访问地址，完成设置后， codis
server 和 codis proxy 才会开始接收连接。

然后，当客户端要读写数据时，客户端直接和 codis proxy 建⽴连接。你
可能会担⼼，既然客户端连接的是 proxy，是不是需要修改客户端，才能
访问 proxy？其实，你不⽤担⼼， codis proxy 本身⽀持 Redis 的 RESP
交互协议，所以，客户端访问 codis proxy 时，和访问原⽣的 Redis 实
例没有什么区别，这样⼀来，原本连接单实例的客户端就可以轻松地和
Codis 集群建⽴起连接了。

最后， codis proxy 接收到请求，就会查询请求数据和 codis server 的映
射关系，并把请求转发给相应的 codis server 进⾏处理。当 codis
server 处理完请求后，会把结果返回给 codis proxy， proxy 再把数据返
回给客户端。

这张图展示了处理流程：





![codis-request-process.png](images%2Fcodis-request-process.png)





好了，了解了 Codis 集群架构和基本流程后，接下来，围绕影响切
⽚集群使⽤效果的 4 ⽅⾯技术因素：数据分布、集群扩容和数据迁移、
客户端兼容性、可靠性保证，讲一下它们的具体设计选择和原理。


# 2 codis的关键技术原理

⼀旦我们使⽤了切⽚集群，⾯临的第⼀个问题就是， 数据是怎么在多个
实例上分布的。


## 2.1 数据如何在集群⾥分布？

在 Codis 集群中，⼀个数据应该保存在哪个 codis server 上，这是通过
逻辑槽（Slot）映射来完成的，具体来说，总共分成两步。

第⼀步， Codis 集群⼀共有 1024 个 Slot，编号依次是 0 到 1023。我们
可以把这些 Slot ⼿动分配给 codis server，每个 server 上包含⼀部分
Slot。当然，我们也可以让 codis dashboard 进⾏⾃动分配，例如，
dashboard 把 1024 个 Slot 在所有 server 上均分。

第⼆步，当客户端要读写数据时，会使⽤ CRC32 算法计算数据 key 的哈
希值，并把这个哈希值对 1024 取模。⽽取模后的值，则对应 Slot 的编
号。此时，根据第⼀步分配的 Slot 和 server 对应关系，我们就可以知道
数据保存在哪个 server 上了。

举个例⼦。下图显示的就是数据、 Slot 和 codis server 的映射保存
关系。其中， Slot 0 和 1 被分配到了 server1， Slot 2 分配到 server2，
Slot 1022 和 1023 被分配到 server8。当客户端访问 key 1 和 key 2 时，
这两个数据的 CRC32 值对 1024 取模后，分别是 1 和 1022。因此，它们
会被保存在 Slot 1 和 Slot 1022 上，⽽ Slot 1 和 Slot 1022 已经被分配到
codis server 1 和 8 上了。这样⼀来， key 1 和 key 2 的保存位置就很清
楚了。





![codis-data-map-to-server.png](images%2Fcodis-data-map-to-server.png)






数据 key 和 Slot 的映射关系是客户端在读写数据前直接通过 CRC32 计
算得到的，⽽ Slot 和 codis server 的映射关系是通过分配完成的，所以
就需要⽤⼀个存储系统保存下来，否则，如果集群有故障了，映射关系就会丢失。

我们把 Slot 和 codis server 的映射关系称为数据路由表（简称路由
表）。我们在 codis dashboard 上分配好路由表后， dashboard 会把路
由表发送给 codis proxy，同时， dashboard 也会把路由表保存在
Zookeeper 中。 codis-proxy 会把路由表缓存在本地，当它接收到客户
端请求后，直接查询本地的路由表，就可以完成正确的请求转发了。

你可以看下这张图，它显示了路由表的分配和使⽤过程。





![codis-route-distribution-and-usage.png](images%2Fcodis-route-distribution-and-usage.png)





在数据分布的实现⽅法上， Codis 和 Redis Cluster 很相似，都采⽤了
key 映射到 Slot、 Slot 再分配到实例上的机制。

但是，这⾥有⼀个明显的区别，我来解释⼀下。

Codis 中的路由表是我们通过 codis dashboard 分配和修改的，并被保
存在 Zookeeper 集群中。⼀旦数据位置发⽣变化（例如有实例增减），
路由表被修改了， codis dashbaord 就会把修改后的路由表发送给 codis
proxy， proxy 就可以根据最新的路由信息转发请求了。

在 Redis Cluster 中，数据路由表是通过每个实例相互间的通信传递的，
最后会在每个实例上保存⼀份。当数据路由信息发⽣变化时，就需要在
所有实例间通过⽹络消息进⾏传递。所以，如果实例数量较多的话，就
会消耗较多的集群⽹络资源。

数据分布解决了新数据写⼊时该保存在哪个 server 的问题，但是，当业
务数据增加后，如果集群中的现有实例不⾜以保存所有数据，我们就需
要对集群进⾏扩容。接下来，我们再来学习下 Codis 针对集群扩容的关
键技术设计。

## 2.2 集群扩容和数据迁移如何进⾏?

Codis 集群扩容包括了两⽅⾯：增加 codis server 和增加 codis proxy。

我们先来看增加 codis server，这个过程主要涉及到两步操作：

1. 启动新的 codis server，将它加⼊集群；
2. 把部分数据迁移到新的 server。

需要注意的是， 这⾥的数据迁移是⼀个重要的机制，接下来我来重点介绍下。

Codis 集群按照 Slot 的粒度进⾏数据迁移，我们来看下迁移的基本流程。

1. 在源 server 上， Codis 从要迁移的 Slot 中随机选择⼀个数据，
   发送给⽬的 server。
2. ⽬的 server 确认收到数据后，会给源 server 返回确认消息。这
   时，源 server 会在本地将刚才迁移的数据删除。
3. 第⼀步和第⼆步就是单个数据的迁移过程。 Codis 会不断重复这
   个迁移过程，直到要迁移的 Slot 中的数据全部迁移完成。

下⾯这张图，显示了数据迁移的流程:






![codis-single-slot-migrate.png](images%2Fcodis-single-slot-migrate.png)





针对刚才介绍的单个数据的迁移过程， Codis 实现了两种迁移模式，分别
是同步迁移和异步迁移，我们来具体看下。

同步迁移是指，在数据从源 server 发送给⽬的 server 的过程中，源
server 是阻塞的，⽆法处理新的请求操作。这种模式很容易实现，但是
迁移过程中会涉及多个操作（包括数据在源 server 序列化、⽹络传输、
在⽬的 server 反序列化，以及在源 server 删除），如果迁移的数据是⼀
个 bigkey，源 server 就会阻塞较⻓时间，⽆法及时处理⽤户请求。

为了避免数据迁移阻塞源 server， Codis 实现的第⼆种迁移模式就是异
步迁移。异步迁移的关键特点有两个。

第⼀个特点是，当源 server 把数据发送给⽬的 server 后，就可以处理其
他请求操作了，不⽤等到⽬的 server 的命令执⾏完。⽽⽬的 server 会在
收到数据并反序列化保存到本地后，给源 server 发送⼀个 ACK 消息，表
明迁移完成。此时，源 server 在本地把刚才迁移的数据删除。

在这个过程中，迁移的数据会被设置为只读，所以，源 server 上的数据
不会被修改，⾃然也就不会出现“和⽬的 server 上的数据不⼀致”问题了。

第⼆个特点是，对于 bigkey，异步迁移采⽤了拆分指令的⽅式进⾏迁
移。具体来说就是，对 bigkey 中每个元素，⽤⼀条指令进⾏迁移，⽽不
是把整个 bigkey 进⾏序列化后再整体传输。这种化整为零的⽅式，就避
免了 bigkey 迁移时，因为要序列化⼤量数据⽽阻塞源 server 的问题。

此外，当 bigkey 迁移了⼀部分数据后，如果 Codis 发⽣故障，就会导致
bigkey 的⼀部分元素在源 server，⽽另⼀部分元素在⽬的 server，这就
破坏了迁移的原⼦性。

所以， Codis 会在⽬标 server 上，给 bigkey 的元素设置⼀个临时过期时
间。如果迁移过程中发⽣故障，那么，⽬标 server 上的 key 会在过期后
被删除，不会影响迁移的原⼦性。当正常完成迁移后， bigkey 元素的临
时过期时间会被删除。

我给你举个例⼦，假如我们要迁移⼀个有 1 万个元素的 List 类型数据，
当使⽤异步迁移时，源 server 就会给⽬的 server 传输 1 万条 RPUSH 命
令，每条命令对应了 List 中⼀个元素的插⼊。在⽬的 server 上，这 1 万
条命令再被依次执⾏，就可以完成数据迁移。

这⾥，有个地⽅需要你注意下，为了提升迁移的效率， Codis 在异步迁移
Slot 时，允许每次迁移多个 key。 你可以通过异步迁移命令
SLOTSMGRTTAGSLOT-ASYNC 的参数 numkeys 设置每次迁移的 key数量。

刚刚我们学习的是 codis server 的扩容和数据迁移机制，其实，在
Codis 集群中，除了增加 codis server，有时还需要增加 codis proxy。

因为在 Codis 集群中，客户端是和 codis proxy 直接连接的，所以，当
客户端增加时，⼀个 proxy ⽆法⽀撑⼤量的请求操作，此时，我们就需
要增加 proxy。

增加 proxy ⽐较容易，我们直接启动 proxy，再通过 codis dashboard
把 proxy 加⼊集群就⾏。

此时， codis proxy 的访问连接信息都会保存在 Zookeeper 上。所以，
当新增了 proxy 后， Zookeeper 上会有最新的访问列表，客户端也就可
以从 Zookeeper 上读取 proxy 访问列表，把请求发送给新增的 proxy。
这样⼀来，客户端的访问压⼒就可以在多个 proxy 上分担处理了，如下
图所示：





![codis-add-proxy.png](images%2Fcodis-add-proxy.png)





好了，到这⾥，我们就了解了 Codis 集群中的数据分布、集群扩容和数
据迁移的⽅法，这都是切⽚集群中的关键机制。

不过，因为集群提供的功能和单实例提供的功能不同，所以，我们在应
⽤集群时，不仅要关注切⽚集群中的关键机制，还需要关注客户端的使
⽤。这⾥就有⼀个问题了：业务应⽤采⽤的客户端能否直接和集群交互
呢？接下来，我们就来聊下这个问题。


## 2.3 集群客户端需要重新开发吗?

使⽤ Redis 单实例时，客户端只要符合 RESP 协议，就可以和实例进⾏
交互和读写数据。但是，在使⽤切⽚集群时，有些功能是和单实例不⼀
样的，⽐如集群中的数据迁移操作，在单实例上是没有的，⽽且迁移过
程中，数据访问请求可能要被重定向（例如 Redis Cluster 中的 MOVE
命令）。

所以，客户端需要增加和集群功能相关的命令操作的⽀持。如果原来使
⽤单实例客户端，想要扩容使⽤集群，就需要使⽤新客户端，这对于业
务应⽤的兼容性来说，并不是特别友好。

Codis 集群在设计时，就充分考虑了对现有单实例客户端的兼容性。

Codis 使⽤ codis proxy 直接和客户端连接， codis proxy 是和单实例客
户端兼容的。⽽和集群相关的管理⼯作（例如请求转发、数据迁移
等），都由 codis proxy、 codis dashboard 这些组件来完成，不需要客
户端参与。

这样⼀来，业务应⽤使⽤ Codis 集群时，就不⽤修改客户端了，可以复
⽤和单实例连接的客户端，既能利⽤集群读写⼤容量数据，⼜避免了修
改客户端增加复杂的操作逻辑，保证了业务代码的稳定性和兼容性。
最后，我们再来看下集群可靠性的问题。可靠性是实际业务应⽤的⼀个
核⼼要求。 对于⼀个分布式系统来说，它的可靠性和系统中的组件个数
有关：组件越多，潜在的⻛险点也就越多。和 Redis Cluster 只包含
Redis 实例不⼀样， Codis 集群包含的组件有 4 类。那你就会问了，这么
多组件会降低 Codis 集群的可靠性吗？

## 2.4 怎么保证集群可靠性？

我们来分别看下 Codis 不同组件的可靠性保证⽅法。

⾸先是 codis server。

codis server 其实就是 Redis 实例，只不过增加了和集群操作相关的命
令。 Redis 的主从复制机制和哨兵机制在 codis server 上都是可以使⽤
的，所以， Codis 就使⽤主从集群来保证 codis server 的可靠性。简单
来说就是， Codis 给每个 server 配置从库，并使⽤哨兵机制进⾏监控，
当发⽣故障时，主从库可以进⾏切换，从⽽保证了 server 的可靠性。

在这种配置情况下，每个 server 就成为了⼀个 server group，每个
group 中是⼀主多从的 server。数据分布使⽤的 Slot，也是按照 group

的粒度进⾏分配的。同时， codis proxy 在转发请求时，也是按照数据所
在的 Slot 和 group 的对应关系，把写请求发到相应 group 的主库，读请
求发到 group 中的主库或从库上。

下图展示的是配置了 server group 的 Codis 集群架构。在 Codis 集群
中，我们通过部署 server group 和哨兵集群，实现 codis server 的主从
切换，提升集群可靠性。





![codis-server-group.png](images%2Fcodis-server-group.png)





因为 codis proxy 和 Zookeeper 这两个组件是搭配在⼀起使⽤的，所
以，接下来，我们再来看下这两个组件的可靠性。

在 Codis 集群设计时， proxy 上的信息源头都是来⾃ Zookeeper（例如
路由表）。⽽ Zookeeper 集群使⽤多个实例来保存数据，只要有超过半数的
Zookeeper 实例可以正常⼯作， Zookeeper 集群就可以提供服务，也可以
保证这些数据的可靠性。

所以， codis proxy 使⽤ Zookeeper 集群保存路由表，可以充分利⽤
Zookeeper 的⾼可靠性保证来确保 codis proxy 的可靠性，不⽤再做额
外的⼯作了。当 codis proxy 发⽣故障后，直接重启 proxy 就⾏。重启后
的 proxy，可以通过 codis dashboard 从 Zookeeper 集群上获取路由
表，然后，就可以接收客户端请求进⾏转发了。这样的设计，也降低了
Codis 集群本身的开发复杂度。

对于 codis dashboard 和 codis fe 来说，它们主要提供配置管理和管理
员⼿⼯操作，负载压⼒不⼤，所以，它们的可靠性可以不⽤额外进⾏保证了。


# 3 切⽚集群⽅案选择建议

到这⾥， Codis 和 Redis Cluster 这两种切⽚集群⽅案我们就学完了，我
把它们的区别总结在了⼀张表⾥，你可以对⽐看下。





![codis-vs-cluster.png](images%2Fcodis-vs-cluster.png)





最后，在实际应⽤的时候，对于这两种⽅案，我们该怎么选择呢？

1. 从稳定性和成熟度来看， Codis 应⽤得⽐较早，在业界已经有了
   成熟的⽣产部署。虽然 Codis 引⼊了 proxy 和 Zookeeper，增
   加了集群复杂度，但是， proxy 的⽆状态设计和 Zookeeper ⾃
   身的稳定性，也给 Codis 的稳定使⽤提供了保证。⽽ Redis
   Cluster 的推出时间晚于 Codis，相对来说，成熟度要弱于
   Codis，如果你想选择⼀个成熟稳定的⽅案， Codis 更加合适
   些。

2. 从业务应⽤客户端兼容性来看，连接单实例的客户端可以直接连
   接 codis proxy，⽽原本连接单实例的客户端要想连接 Redis
   Cluster 的话，就需要开发新功能。所以，如果你的业务应⽤中
   ⼤量使⽤了单实例的客户端，⽽现在想应⽤切⽚集群的话，建议
   你选择 Codis，这样可以避免修改业务应⽤中的客户端。

3. 从使⽤ Redis 新命令和新特性来看， Codis server 是基于开源
   的 Redis 3.2.8 开发的，所以， Codis 并不⽀持 Redis 后续的开
   源版本中的新增命令和数据类型。另外， Codis 并没有实现开源
   Redis 版本的所有命令，⽐如 BITOP、 BLPOP、 BRPOP，以及
   和与事务相关的 MUTLI、 EXEC 等命令。 Codis 官⽹上列出了
   不被⽀持的命令列表，你在使⽤时记得去核查⼀下。所以，如果
   你想使⽤开源 Redis 版本的新特性， Redis Cluster 是⼀个合适
   的选择。

4. 从数据迁移性能维度来看， Codis 能⽀持异步迁移，异步迁移对
   集群处理正常请求的性能影响要⽐使⽤同步迁移的⼩。所以，如
   果你在应⽤集群时，数据迁移⽐较频繁的话， Codis 是个更合适
   的选择。


# 4 小结

本节学习了 Redis 切⽚集群的 Codis ⽅案。 Codis 集群包含
codis server、 codis proxy、 Zookeeper、 codis dashboard 和 codis fe
这四⼤类组件。我们再来回顾下它们的主要功能。

• codis proxy 和 codis server 负责处理数据读写请求，其中，
codis proxy 和客户端连接，接收请求，并转发请求给 codis
server，⽽ codis server 负责具体处理请求。

• codis dashboard 和 codis fe 负责集群管理，其中， codis
dashboard 执⾏管理操作，⽽ codis fe 提供 Web 管理界⾯。

• Zookeeper 集群负责保存集群的所有元数据信息，包括路由表、
proxy 实例信息等。这⾥，有个地⽅需要你注意，除了使⽤
Zookeeper， Codis 还可以使⽤ etcd 或本地⽂件系统保存元数据
信息。

关于 Codis 和 Redis Cluster 的选型考虑，我从稳定性成熟度、客户端兼
容性、 Redis 新特性使⽤以及数据迁移性能四个⽅⾯提供了建议。

最后，再提供⼀个 Codis 使⽤上的⼩建议：当你有多条业务线要使⽤ Codis 时，
可以启动多个 codis dashboard，每个 dashboard 管理⼀部分 codis server，
同时，再⽤⼀个 dashboard 对应负责⼀个业务线的集群管理，这样，就可以做到⽤
⼀个 Codis 集群实现多条业务线的隔离管理了。