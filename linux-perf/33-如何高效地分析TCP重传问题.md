
---
如何高效地分析TCP重传问题？
---


# 1 什么是TCP 重传？





![tcp-retran-monitor.png](images%2Ftcp-retran-monitor.png)





这是互联网企业普遍都有的 TCP 重传率监控，它是服务器稳定性的一个指标，如果它太高，就像上图中的那些毛刺一样，往往就意味
着服务器不稳定了。那 TCP 重传率究竟表示什么呢？

其实 TCP 重传率是通过解析 /proc/net/snmp 这个文件里的指标计算出来的，这个文件里面和 TCP 有关的关键指标如下：





![tcp-retran-index.png](images%2Ftcp-retran-index.png)





TCP 重传率的计算公式如下：

```shell
retrans = (RetransSegs－last RetransSegs) ／ (OutSegs－last OutSegs) * 100
```

也就是说，单位时间内 TCP 重传包的数量除以 TCP 总的发包数量，就是 TCP 重传率。那我们继续看下这个公式中的 RetransSegs 
和 OutSegs 是怎么回事，下面两张示例图可以演示这两个指标的变化：





![no-retran-situation.png](images%2Fno-retran-situation.png)





![has-retran-situation.png](images%2Fhas-retran-situation.png)





通过这两个示例图，你可以发现，发送端在发送一个 TCP 数据包后，会把该数据包放在发送端的发送队列里，也叫重传队列。此时，
OutSegs 会相应地加 1，队列长度也为 1。如果可以收到接收端对这个数据包的 ACK，该数据包就会在发送队列中被删掉，然后队列长
度变为 0；如果收不到这个数据包的 ACK，就会触发重传机制，我们在这里演示的就是超时重传这种情况，也就是说发送端在发送数据包
的时候，会启动一个超时重传定时器(RTO），如果超过了这个时间，发送端还没有收到 ACK，就会重传该数据包，然后OutSegs 加 1，
同时 RetransSegs 也会加1。

这就是 OutSegs 和 RetransSegs 的含义：每发出去一个 TCP 包（包括重传包）， OutSegs 会相应地加 1；每发出去一个重传包，
RetransSegs 会相应地加 1。


# 2 哪些情况会导致TCP 重传？

引起 TCP 重传的情况在整体上可以分为如下两类。

## 2.1 丢包
TCP 数据包在网络传输过程中可能会被丢弃；接收端也可能会把该数据包给丢弃；接收端回的 ACK 也可能在网络传输过程中被丢弃；
数据包在传输过程中发生错误而被接收端给丢弃……这些情况都会导致发送端重传该 TCP 数据包。

## 2.2 拥塞
TCP 数据包在网络传输过程中可能会在某个交换机 / 路由器上排队，比如臭名昭著的Bufferbloat（缓冲膨胀）；TCP 数据包在
网络传输过程中因为路由变化而产生的乱序；接收端回的 ACK 在某个交换机 / 路由器上排队……这些情况都会导致发送端再次重传该
TCP 数据包。

总之，TCP 重传可以很好地作为通信质量的信号，我们需要去重视它。
那么，当我们通过监控发现某个主机上 TCP 重传率很高时，该如何去分析呢？


# 3 如何分析 TCP 重传？

## 3.1 分析 TCP 重传的常规手段
最常规的分析手段就是 tcpdump，我们可以使用它把进出某个网卡的数据包给保存下来：
```shell
tcpdump -s 0 -i eth0 -w tcpdumpfile
```

然后在 Linux 上我们可以使用 tshark 这个工具（wireshark 的 Linux 版本）来过滤出 TCP 重传包：

```shell
tshark -r tcpdumpfile -R tcp.analysis.retransmission
```

如果有重传包的话，就可以显示出来了，如下是一个 TCP 重传的示例：

```shell
3481 20.277303 10.17.130.20 -> 124.74.250.144 TCP 70 [TCP Retransmission] 35993 > https [SYN] Seq=0 Win=14600 Len=0 MSS=1460 SACK_PERM=1 TSval=3231504691 TSecr=0
3659 22.277070 10.17.130.20 -> 124.74.250.144 TCP 70 [TCP Retransmission] 35993 > https [SYN] Seq=0 Win=14600 Len=0 MSS=1460 SACK_PERM=1 TSval=3231506691 TSecr=0 
8649 46.539393 58.216.21.165 -> 10.17.130.20 TLSv1 113 [TCP Retransmission] Change Spec, Encrypted Handshake Message
```

借助 tcpdump，我们就可以看到 TCP 重传的详细情况。从上面这几个 TCP 重传信息中，我们可以看到，这是发生在 10.17.130.20:35993 - 124.74.250.144: 443 
这个 TCP 连接上的重传；通过[SYN]这个 TCP 连接状态，可以看到这是发生在第一次握手阶段的重传。依据这些信息，我们就可以继续去 124.74.250.144 这个主机上
分析 https 这个服务为什么无法建立新的连接。
但是，我们都知道 tcpdump 很重，如果直接在生产环境上进行采集的话，难免会对业务造成性能影响。那有没有更加轻量级的一些分析方法呢？


## 3.2 如何高效地分析 TCP 重传 ？

其实，就像应用程序实现一些功能需要调用对应的函数一样，TCP 重传也需要调用特定的内核函数。这个内核函数就是 tcp_retransmit_skb()。
你可以把这个函数名字里的 skb 理解为是一个需要发送的网络包。那么，如果我们想要高效地追踪 TCP 重传情况，那么直接追踪该函数就可以了。
追踪内核函数最通用的方法是使用 Kprobe，Kprobe 的大致原理如下：





![kprobe-demo.png](images%2Fkprobe-demo.png)





你可以实现一个内核模块，该内核模块中使用 Kprobe 在 tcp_retransmit_skb 这个函数入口插入一个 probe，然后注册一个 break_handler，
这样在执行到 tcp_retransmit_skb 时就会异常跳转到注册的 break_handler 中，然后在 break_handler 中解析 TCP 报文 （skb）就可以了，
从而来判断是什么在重传。


Kprobe 这种方式使用起来还是略有些不便，为了让 Linux 用户更方便地观察 TCP 重传事件，4.16 内核版本中专门添加了TCP tracepoint来解析
TCP 重传事件。如果你使用的操作系统是 CentOS-7 以及更老的版本，就无法使用该 Tracepoint 来观察了；如果你的版本是 CentOS-8 以及
后续更新的版本，那你可以直接使用这个 Tracepoint 来追踪 TCP 重传，可以使用如下命令：

```shell
cd /sys/kernel/debug/tracing/events/
echo 1 > tcp/tcp_retransmit_skb/enable
```

然后你就可以追踪 TCP 重传事件了：

```shell
cat trace_pipe
<idle>-0 [007] ..s. 265119.290232: tcp_retransmit_skb: sport=22 dport=62264 ...
```

可以看到，当 TCP 重传发生时，该事件的基本信息就会被打印出来。

追踪结束后呢，你需要将这个 Tracepoint 给关闭：

```shell
echo 0 > tcp/tcp_retransmit_skb/enable
```

Tracepoint 这种方式不仅使用起来更加方便，而且它的性能开销比 Kprobe 要小，所以我们在快速路径上也可以使用它。