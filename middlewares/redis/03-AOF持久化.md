
---
AOF⽇志：宕机了，Redis如何避免数据丢失？
---

如果有⼈问你：“你会把Redis⽤在什么业务场景下？”我想你⼤概率会说：“我会把它当作缓存使⽤，因
为它把后端数据库中的数据存储在内存中，然后直接从内存中读取数据，响应速度会⾮常快。”没错，这确
实是Redis的⼀个普遍使⽤场景，但是，这⾥也有⼀个绝对不能忽略的问题：⼀旦服务器宕机，内存中的数
据将全部丢失。

我们很容易想到的⼀个解决⽅案是，从后端数据库恢复这些数据，但这种⽅式存在两个问题：⼀是，需要频
繁访问数据库，会给数据库带来巨⼤的压⼒；⼆是，这些数据是从慢速数据库中读取出来的，性能肯定⽐不
上从Redis中读取，导致使⽤这些数据的应⽤程序响应变慢。所以，对Redis来说，实现数据的持久化，避免
从后端数据库中进⾏恢复，是⾄关重要的。

⽬前，Redis的持久化主要有两⼤机制，即AOF⽇志和RDB快照。我们先重点学习下AOF⽇志。

# 1 AOF日志是如何实现的？

说到⽇志，我们⽐较熟悉的是数据库的写前⽇志（Write Ahead Log, WAL），也就是说，在实际写数据前，
先把修改的数据记到⽇志⽂件中，以便故障时进⾏恢复。不过，AOF⽇志正好相反，它是写后⽇志，“写
后”的意思是Redis是先执⾏命令，把数据写⼊内存，然后才记录⽇志，如下图所⽰:





![redis-write-behind-log.png](images%2Fredis-write-behind-log.png)





那AOF为什么要先执⾏命令再记⽇志呢？要回答这个问题，我们要先知道AOF⾥记录了什么内容。

传统数据库的⽇志，例如redo log（重做⽇志），记录的是修改后的数据，⽽AOF⾥记录的是Redis收到的
每⼀条命令，这些命令是以⽂本形式保存的。

我们以Redis收到“set testkey testvalue”命令后记录的⽇志为例，看看AOF⽇志的内容。其中，“*3”表
⽰当前命令有三个部分，每部分都是由“$+数字”开头，后⾯紧跟着具体的命令、键或值。这⾥，“数
字”表⽰这部分中的命令、键或值⼀共有多少字节。例如，“$3 set”表⽰这部分有3个字节，也就是“set”命令。





![aof-file.png](images%2Faof-file.png)





但是，为了避免额外的检查开销，Redis在向AOF⾥⾯记录⽇志的时候，并不会先去对这些命令进⾏语法检
查。所以，如果先记⽇志再执⾏命令的话，⽇志中就有可能记录了错误的命令，Redis在使⽤⽇志恢复数据
时，就可能会出错。

⽽写后⽇志这种⽅式，就是先让系统执⾏命令，只有命令能执⾏成功，才会被记录到⽇志中，否则，系统就
会直接向客⼾端报错。所以，Redis使⽤写后⽇志这⼀⽅式的⼀⼤好处是，可以避免出现记录错误命令的情况。

除此之外，AOF还有⼀个好处：它是在命令执⾏后才记录⽇志，所以不会阻塞当前的写操作。
不过，AOF也有两个潜在的⻛险。

⾸先，如果刚执⾏完⼀个命令，还没有来得及记⽇志就宕机了，那么这个命令和相应的数据就有丢失的⻛险。
如果此时Redis是⽤作缓存，还可以从后端数据库重新读⼊数据进⾏恢复，但是，如果Redis是直接⽤作
数据库的话，此时，因为命令没有记⼊⽇志，所以就⽆法⽤⽇志进⾏恢复了。

其次，AOF虽然避免了对当前命令的阻塞，但可能会给下⼀个操作带来阻塞⻛险。这是因为，AOF⽇志也是
在主线程中执⾏的，如果在把⽇志⽂件写⼊磁盘时，磁盘写压⼒⼤，就会导致写盘很慢，进⽽导致后续的操
作也⽆法执⾏了。

仔细分析的话，你就会发现，这两个⻛险都是和AOF写回磁盘的时机相关的。这也就意味着，如果我们能够
控制⼀个写命令执⾏完后AOF⽇志写回磁盘的时机，这两个⻛险就解除了。

# 2 三种写回策略

其实，对于这个问题，AOF机制给我们提供了三个选择，也就是AOF配置项appendfsync的三个可选值。

Always，同步写回：每个写命令执⾏完，⽴⻢同步地将⽇志写回磁盘；
Everysec，每秒写回：每个写命令执⾏完，只是先把⽇志写到AOF⽂件的内存缓冲区，每隔⼀秒把缓冲
区中的内容写⼊磁盘；
No，操作系统控制的写回：每个写命令执⾏完，只是先把⽇志写到AOF⽂件的内存缓冲区，由操作系统
决定何时将缓冲区内容写回磁盘。

针对避免主线程阻塞和减少数据丢失问题，这三种写回策略都⽆法做到两全其美。我们来分析下其中的原因。

“同步写回”可以做到基本不丢数据，但是它在每⼀个写命令后都有⼀个慢速的落盘操作，不可避免地会
影响主线程性能；
虽然“操作系统控制的写回”在写完缓冲区后，就可以继续执⾏后续的命令，但是落盘的时机已经不在
Redis⼿中了，只要AOF记录没有写回磁盘，⼀旦宕机对应的数据就丢失了；
“每秒写回”采⽤⼀秒写回⼀次的频率，避免了“同步写回”的性能开销，虽然减少了对系统性能的影
响，但是如果发⽣宕机，上⼀秒内未落盘的命令操作仍然会丢失。所以，这只能算是，在避免影响主线程
性能和避免数据丢失两者间取了个折中。

这三种策略的写回时机，以及优缺点汇总如下表所示:





![aof-write-back-strategy.png](images%2Faof-write-back-strategy.png)



刷盘（fsync 或 fdatasync）是关键的一步，它负责将内存中的数据同步到磁盘文件，确保数据持久化。

always 模式下，fsync 是由主线程同步执行的，无法交由后台线程异步执行，因此会阻塞主线程并影响写入性能。
everysec 模式下，fsync 是由后台线程异步执行的，不会阻塞主线程，因此对性能影响较小。
no 模式下，fsync 完全由操作系统的调度策略决定，主线程完全没有刷盘开销，性能最佳，但数据丢失风险最高。

到这⾥，我们就可以根据系统对⾼性能和⾼可靠性的要求，来选择使⽤哪种写回策略了。总结⼀下就是：想
要获得⾼性能，就选择No策略；如果想要得到⾼可靠性保证，就选择Always策略；如果允许数据有⼀点丢
失，⼜希望性能别受太⼤影响的话，那么就选择Everysec策略。

但是，按照系统的性能需求选定了写回策略，并不是“⾼枕⽆忧”了。毕竟，AOF是以⽂件的形式在记录接
收到的所有写命令。随着接收的写命令越来越多，AOF⽂件会越来越⼤。这也就意味着，我们⼀定要⼩⼼
AOF⽂件过⼤带来的性能问题。

这⾥的“性能问题”，主要在于以下三个⽅⾯：⼀是，⽂件系统本⾝对⽂件⼤⼩有限制，⽆法保存过⼤的⽂
件；⼆是，如果⽂件太⼤，之后再往⾥⾯追加命令记录的话，效率也会变低；三是，如果发⽣宕机，AOF中
记录的命令要⼀个个被重新执⾏，⽤于故障恢复，如果⽇志⽂件太⼤，整个恢复过程就会⾮常缓慢，这就会
影响到Redis的正常使⽤。

所以，我们就要采取⼀定的控制⼿段，这个时候，AOF重写机制就登场了。


# 3 AOF重写机制

## 3.1 多变一

简单来说，AOF重写机制就是在重写时，Redis根据数据库的现状创建⼀个新的AOF⽂件，也就是说，读取
数据库中的所有键值对，然后对每⼀个键值对⽤⼀条命令记录它的写⼊。⽐如说，当读取了键值
对“testkey”: “testvalue”之后，重写机制会记录set testkey testvalue这条命令。这样，当需要恢复
时，可以重新执⾏该命令，实现“testkey”: “testvalue”的写⼊。

为什么重写机制可以把⽇志⽂件变⼩呢? 实际上，重写机制具有“多变⼀”功能。所谓的“多变⼀”，也就
是说，旧⽇志⽂件中的多条命令，在重写后的新⽇志中变成了⼀条命令。

我们知道，AOF⽂件是以追加的⽅式，逐⼀记录接收到的写命令的。当⼀个键值对被多条写命令反复修改
时，AOF⽂件会记录相应的多条命令。但是，在重写的时候，是根据这个键值对当前的最新状态，为它⽣成
对应的写⼊命令。这样⼀来，⼀个键值对在重写⽇志中只⽤⼀条命令就⾏了，⽽且，在⽇志恢复时，只⽤执
⾏这条命令，就可以直接完成这个键值对的写⼊了。

下⾯这张图就是⼀个例⼦：





![aof-rewrite-example.png](images%2Faof-rewrite-example.png)





当我们对⼀个列表先后做了6次修改操作后，列表的最后状态是[“D”, “C”, “N”]，此时，只⽤LPUSH
u:list “N”, “C”, "D"这⼀条命令就能实现该数据的恢复，这就节省了五条命令的空间。对于被修改过成
百上千次的键值对来说，重写能节省的空间当然就更⼤了。

不过，虽然AOF重写后，⽇志⽂件会缩⼩，但是，要把整个数据库的最新数据的操作⽇志都写回磁盘，仍然
是⼀个⾮常耗时的过程。这时，我们就要继续关注另⼀个问题了：重写会不会阻塞主线程？


## 3.2 后台线程异步执行aof重写

和AOF⽇志由主线程写回不同，重写过程是由后台线程bgrewriteaof来完成的，这也是为了避免阻塞主线
程，导致数据库性能下降。

重写的过程可以总结为“⼀个拷⻉，两处⽇志”。

“⼀个拷⻉”就是指，每次执⾏重写时，主线程fork出后台的bgrewriteaof⼦进程。此时，fork会把主线程
的内存拷⻉⼀份给bgrewriteaof⼦进程，这⾥⾯就包含了数据库的最新数据。然后，bgrewriteaof⼦进程就
可以在不影响主线程的情况下，逐⼀把拷⻉的数据写成操作，记⼊重写⽇志。

“两处⽇志”⼜是什么呢？因为主线程未阻塞，仍然可以处理新来的操作。此时，如果有写操作，第⼀处⽇志就是指正在使⽤的AOF⽇
志，Redis会把这个操作写到它的缓冲区。这样⼀来，即使宕机了，这个AOF⽇志的操作仍然是⻬全的，可以⽤于恢复。

⽽第⼆处⽇志，就是指新的AOF重写⽇志。这个操作也会被写到重写⽇志的缓冲区。这样，重写⽇志也不会
丢失最新的操作。等到拷⻉数据的所有操作记录重写完成后，重写⽇志记录的这些最新操作也会写⼊新的
AOF⽂件，以保证数据库最新状态的记录。此时，我们就可以⽤新的AOF⽂件替代旧⽂件了。





![aof-process.png](images%2Faof-process.png)





总结来说，每次AOF重写时，Redis会先执⾏⼀个内存拷⻉，⽤于重写；然后，使⽤两个⽇志保证在重写过
程中，新写⼊的数据不会丢失。⽽且，因为Redis采⽤额外的线程进⾏数据重写，所以，这个过程并不会阻
塞主线程。


### 3.3  AOF重写的触发时机

Redis 的 AOF（Append-Only File）重写机制是默认开启的，并且它是基于一定条件触发的。这些条件主要是与
AOF 文件的大小和与当前数据库状态的关系相关：

**AOF重写的触发条件**

AOF 文件增长限制：

当 AOF 文件的增长量达到原有 AOF 文件大小的 100% 时，Redis 会触发 AOF 重写（默认情况下）。
这个条件意味着，如果 Redis 的 AOF 文件越来越大，它会通过重写来减小 AOF 文件的大小，将其中的重复写操作合并，减少文件的冗余。

后台重写（bgrewriteaof）：

AOF 重写是通过 Redis 的 后台进程（子进程） 实现的，主线程会 fork 一个子进程来处理重写操作，而主进程继续响应客户端的请求。
子进程会根据当前 Redis 数据库的状态（包括所有键的当前值）生成一个新的 AOF 文件，并将其重写。
重写过程是增量的，会将所有的写操作命令合并，并消除重复操作，最终生成一个新的更小的 AOF 文件。

AOF重写机制的自动触发：
默认情况下，Redis 会在 AOF 文件大小翻倍时触发重写，即当当前的 AOF 文件大小达到原始文件的两倍时，它会开始触发 AOF 重写。
Redis 配置项 auto-aof-rewrite-percentage 和 auto-aof-rewrite-min-size 控制这一行为：
auto-aof-rewrite-percentage：表示 AOF 文件大小增长的百分比，达到这个阈值时会触发重写。默认值为 100（即文件大小翻倍时触发）。
auto-aof-rewrite-min-size：表示 AOF 文件最小的大小，只有文件超过此大小才会触发重写。默认值为 64MB。


**AOF全量日志和重写AOF日志的选择**

全量 AOF 文件与重写 AOF 文件的差异：

全量 AOF 文件：这是 Redis 持久化所有写操作时产生的 AOF 文件，包含每一个写操作命令，逐条写入，保持完整的操作日志。其特点是文件较大，
包含冗余的重复操作。

重写后的 AOF 文件：重写过程中，Redis 会合并重复的操作，减少冗余的写入命令。也就是说，重写后的 AOF 文件会更加紧凑，通常比全量的 AOF 
文件小得多，并且只包含当前数据库的必要操作命令，而没有重复的历史操作。

恢复数据时使用的AOF 文件：
当 Redis 宕机 后恢复时，Redis 优先使用 重写后的 AOF 文件，而不是全量的 AOF 文件。这是因为重写后的 AOF 文件更为紧凑，包含的是
当前数据库的最新状态，文件较小且不包含重复的命令。它包含了所有必要的写操作，并且性能上也更优。

AOF 全量日志是否有必要存在：
全量 AOF 日志仍然有其存在的必要，主要原因如下：
写入效率：全量 AOF 日志包含所有的写操作，包括重复的操作，而重写过程是增量的。如果 Redis 在某些场景下需要避免复杂的重写过程
（例如，在高并发场景下），全量的 AOF 日志可以确保完整记录所有的操作，防止丢失任何操作。
备份和恢复的兼容性：重写后的 AOF 文件适合于恢复数据库的最新状态，但在某些场景下，全量 AOF 文件对于数据恢复的兼容性更好，尤其是在
需要追溯到某一特定时间点时。

是否需要同时保留全量日志和重写日志：
在实际的 Redis 部署中，如果不考虑 RDB（全量快照）持久化，仅依赖 AOF 日志，系统通常仍然会保留全量 AOF 文件和重写后的 AOF 文件的备份。
但重写后的 AOF 文件的大小相较于全量 AOF 文件要小很多，因此对于长期存储来说，重写 AOF 文件更为高效。