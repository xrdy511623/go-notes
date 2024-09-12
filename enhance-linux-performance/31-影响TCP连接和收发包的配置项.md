
---
影响TCP连接和收发包的配置项
---

# 1 影响TCP连接建立的配置项

## 1.1 tcp_syn_retries 控制第一次握手失败的超时重传次数
在TCP建立连接前的三次握手过程中，如果第一次握手(client向server发送SYN包)后，client一直收不到server返回的
SYN+ACK包，就会触发client的超时重传机制，每次重传的间隔时间翻倍，初始间隔时间是1s，而重传次数是有限制的，
这就是由tcp_syn_retries这个配置项来决定的。

tcp_syn_retries配置项默认值为6，也就是说如果SYN包发出后，一直收不到server的对应SYN+ACK包，会重传6次，在
1+2+4+8+16+32+64=127s后产生TIMEOUT错误。

我们在生产环境上就遇到过这种情况，Server 因为某些原因被下线，但是 Client 没有被通 知到，所以 Client 的 connect() 
被阻塞 127s 才去尝试连接一个新的 Server， 这么长的 超时等待时间对于应用程序而言是很难接受的。
所以通常情况下，我们都会将数据中心内部服务器的 tcp_syn_retries 给调小，这里推荐 设置为 2，来减少阻塞的时间。
因为对于数据中心而言，它的网络质量是很好的，如果得 不到 Server 的响应，很可能是 Server 本身出了问题。在这种情况下，
Client 及早地去尝 试连接其他的 Server 会是一个比较好的选择，所以对于客户端而言，一般都会做如下调整:

```shell
net.ipv4.tcp_syn_retries = 2
```

有些情况下 1s 的阻塞时间可能都很久，所以有的时候也会将三次握手的初始超时时间从默 认值 1s 调整为一个较小的值，比如
100ms，这样整体的阻塞时间就会小很多。这也是数 据中心内部经常进行一些网络优化的原因。


## 1.2 tcp_max_syn_backlog 防止半连接队列溢出
如果 Server 没有响应 Client 的 SYN，除了 Server 已经不存在了这种情况外，还有可能是因为 Server 太忙没有来得及响应，
或者是 Server 已经积压了太多的半连接(incomplete)而无法及时去处理。
半连接，即收到了 SYN 后还没有回复 SYN+ACK 的连接，Server 每收到一个新的 SYN 包，都会创建一个半连接，然后把该半连接
加入到半连接队列(syn queue)中。syn queue 的长度就是 tcp_max_syn_backlog 这个配置项来决定的，当系统中积压的半连接
个数超过了该值后，新的 SYN 包就会被丢弃。对于服务器而言，可能瞬间会有非常多的新 建连接，所以我们可以适当地调大该值，以免
SYN 包被丢弃而导致 Client 收不到 SYNACK:

```shell
net.ipv4.tcp_max_syn_backlog = 16384
```

## 1.3 tcp_syncookies  防止SYN Flood攻击
Server 中积压的半连接较多，也有可能是因为有些恶意的 Client 在进行 SYN Flood 攻 击。典型的 SYN Flood 攻击如下:
Client 高频地向 Server 发 SYN 包，并且这个 SYN 包 的源 IP 地址不停地变换，那么 Server 每次接收到一个新的 SYN 后，
都会给它分配一个半 连接，Server 的 SYNACK 根据之前的 SYN 包找到的是错误的 Client IP， 所以也就无法收到 Client 的
ACK 包，导致无法正确建立 TCP 连接，这就会让 Server 的半连接队列耗尽，无法响应正常的 SYN 包。

为了防止 SYN Flood 攻击，Linux 内核引入了 SYN Cookies 机制。SYN Cookie 的原理 是什么样的呢?
在 Server 收到 SYN 包时，不去分配资源来保存 Client 的信息，而是根据这个 SYN 包计 算出一个 Cookie 值，然后将
Cookie 记录到 SYN+ACK 包中发送出去。对于正常的连接， 该 Cookies 值会随着 Client 的 ACK 报文被带回来。然后
Server 再根据这个 Cookie 检查 这个 ACK 包的合法性，如果合法，才去创建新的 TCP 连接。通过这种处理，
SYN Cookies 可以防止部分 SYN Flood 攻击。所以对于 Linux 服务器而言，推荐开启 SYN Cookies:

```shell
net.ipv4.tcp_syncookies = 1
```


## 1.4 tcp_synack_retries 控制第二次握手失败的超时重传次数 
Server 向 Client 发送的 SYN+ACK 包也可能会被丢弃，或者因为某些原因而收不到 Client 的响应，这个时候 Server 也会重传
SYN+ACK 包。同样地，重传的次数也是由配置选项来控制的，该配置选项是 tcp_synack_retries。
tcp_synack_retries 的重传策略跟我们在前面讲的 tcp_syn_retries 是一致的，它在系统中默认是 5，对于数据中心的服务器而言，
通常都不需要这么大的值，推荐设置为 2 :

```shell
net.ipv4.tcp_synack_retries = 2
```

## 1.5 somaxconn 防止全连接队列溢出
Client 在收到 Server 的 SYN+ACK 包后，就会发出 ACK，Server 收到该 ACK 后，三次握 手就完成了，即产生了一个 TCP 
全连接(complete)，它会被添加到全连接队列 (accept queue)中。然后 Server 就会调用 accept() 来完成 TCP 连接的建立。
但是，就像半连接队列(syn queue)的长度有限制一样，全连接队列(accept queue) 的长度也有限制，目的就是为了防止 Server 
不能及时调用 accept() 而浪费太多的系统资源。

全连接队列(accept queue)的长度是由 listen(sockfd, backlog) 这个函数里的 backlog 控制的，而该 backlog 的最大值
则是 somaxconn。somaxconn 在 5.4 之前的内核中， 默认都是 128(5.4 开始调整为了默认 4096)，建议将该值适当调大一些:

```shell
net.core.somaxconn = 16384
```

## 1.6 tcp_abort_on_overflow 全连接队列满时是否需要发送reset包
当服务器中积压的全连接个数超过该值后，新的全连接就会被丢弃掉。Server 在将新连接丢弃时，有的时候需要发送 reset 来通知
Client，这样 Client 就不会再次重试了。不过， 默认行为是直接丢弃不去通知 Client。至于是否需要给 Client 发送 reset，
是由 tcp_abort_on_overflow 这个配置项来控制的，该值默认为 0，即不发送 reset 给 Client。推荐也是将该值配置为 0:

```shell
net.ipv4.tcp_abort_on_overflow = 0
```

这是因为，Server 如果来不及 accept() 而导致全连接队列满，这往往是由瞬间有大量新建 连接请求导致的，正常情况下 Server
很快就能恢复，然后 Client 再次重试后就可以建连 成功了。也就是说，将 tcp_abort_on_overflow 配置为 0，给了 
Client 一个重试的机会。 当然，你可以根据你的实际情况来决定是否要使能该选项。

accept() 成功返回后，一个新的 TCP 连接就建立完成了，TCP 连接进入到了 ESTABLISHED 状态:





![tcp-three-times-handshake.png](images%2Ftcp-three-times-handshake.png)





上图就是从 Client 调用 connect()，到 Server 侧 accept() 成功返回这一过程中的 TCP 状 态转换。这些状态都可以通过
netstat 或者 ss 命令来看。至此，Client 和 Server 两边就 可以正常通信了。


# 2 影响TCP连接断开的配置项





![tcp-four-times-wave.png](images%2Ftcp-four-times-wave.png)





如上图所示，客户端打算关闭连接，此时会发送⼀个 TCP首部 FIN 标志位被置为 1 的报⽂文，也即 FIN 报文，之后客户端进⼊
FIN_WAIT_1 状态。
服务端收到该报⽂后，就向客户端发送 ACK 应答报文，接着服务端进⼊ CLOSED_WAIT 状态。 
客户端收到服务端的 ACK 应答报文后，之后进⼊入 FIN_WAIT_2 状态。 
等待服务端处理理完数据后，也向客户端发送 FIN 报文，之后服务端进⼊ LAST_ACK 状态。 
客户端收到服务端的 FIN 报⽂后，回⼀个 ACK 应答报文，之后进⼊ TIME_WAIT 状态
服务器器收到了 ACK 应答报⽂后，就进⼊了 CLOSED 状态，⾄至此服务端已经完成连接的关闭。 
客户端在经过 2MSL 一段时间后，自动进⼊ CLOSED 状态，⾄至此客户端也完成连接的关闭。
你可以看到，每个⽅向都需要⼀个 FIN 和一个 ACK，因此通常被称为四次挥⼿手。 这⾥一点需要注意是:主动关闭连接的，
才有 TIME_WAIT 状态。


## 2.1  tcp_fin_timeout 控制处于FIN_WAIT_2状态的超时时间
TCP 进入到这个状态后，如果本端迟迟收不到对端的 FIN 包，那就会一直处于这个状态，于是就会一直消耗系统资源。Linux 为了防止这种资源的开销，
设置了这个状态的超时时间 tcp_fin_timeout，默认为 60s，超过这个时间后就会自动销毁该连接。
至于本端为何迟迟收不到对端的 FIN 包，通常情况下都是因为对端机器出了问题，或者是 因为太繁忙而不能及时 close()。所以，通常我们都建议将
tcp_fin_timeout 调小一些，以尽量避免这种状态下的资源开销。对于数据中心内部的机器而言，将它调整为 2s 足矣:

```shell
net.ipv4.tcp_fin_timeout = 2
```

## 2.2  tcp_max_tw_buckets 控制处于TIME_WAIT 状态的连接数
TIME_WAIT 状态存在的意义是:最后发送的这个 ACK 包可能会被丢弃掉或者有延迟，这样对端就会再次发送 FIN 包。如果不维持 TIME_WAIT 这个状态，
那么再次收到对端的 FIN 包后，本端就会回一个 Reset 包，这可能会产生一些异常。
所以维持 TIME_WAIT 状态一段时间，可以保障 TCP 连接正常断开。TIME_WAIT 的默认 存活时间在 Linux 上是 60s(TCP_TIMEWAIT_LEN)，
这个时间对于数据中心而言可能还是有些长了，所以有的时候也会修改内核做些优化来减小该值，或者将该值设置为可通过 sysctl 来调节。
TIME_WAIT 状态存在这么长时间，也是对系统资源的一个浪费，所以系统也有配置项来限制该状态的最大个数，该配置选项就是 tcp_max_tw_buckets。
对于数据中心而言，网络是相对很稳定的，基本不会存在 FIN 包的异常，所以建议将该值调小一些:

```shell
net.ipv4.tcp_max_tw_buckets = 10000
```

## 2.3 tcp_tw_reuse 允许复用处于 TIME_WAIT 状态的连接
Client 关闭跟 Server 的连接后，也有可能很快再次跟 Server 之间建立一个新的连接，而由于 TCP 端口最多只有 65536 个，如果不去
复用处于 TIME_WAIT 状态的连接，就可能在快速重启应用程序时，出现端口被占用而无法创建新连接的情况。所以建议你打开复用 TIME_WAIT 的选项:

```shell
net.ipv4.tcp_tw_reuse = 1
```

# 3 TCP 数据包的发送过程会受什么影响？





![tcp-send-packet-process.png](images%2Ftcp-send-packet-process.png)





## 3.1 tcp_wmem 和 wmem_max，控制单个TCP 连接发送缓冲区的大小
上图就是一个简略的 TCP 数据包的发送过程。应用程序调用 write(2) 或者 send(2) 系统调用开始往外发包时，这些系统调用
会把数据包从用户缓冲区拷贝到 TCP 发送缓冲区 （TCP Send Buffer），这个 TCP 发送缓冲区的大小是受限制的，这里也是容易引起问题
的地方。

TCP 发送缓冲区的大小默认是受 net.ipv4.tcp_wmem 来控制：

```shell
net.ipv4.tcp_wmem = 8192 65536 16777216
```

tcp_wmem 中这三个数字的含义分别为 min、default、max。TCP 发送缓冲区的大小会 在 min 和 max 之间动态调整，初始的大小是 
default，这个动态调整的过程是由内核自动来做的，应用程序无法干预。自动调整的目的，是为了在尽可能少地浪费内存的情况下来满足发包的需要。
tcp_wmem 中的 max 不能超过 net.core.wmem_max 这个配置项的值，如果超过了， TCP 发送缓冲区最大就是 net.core.wmem_max。
通常情况下，我们需要设置 net.core.wmem_max 的值大于等于 net.ipv4.tcp_wmem 的 max:

```shell
net.core.wmem_max = 16777216
```

对于 TCP 发送缓冲区的大小，我们需要根据服务器的负载能力来灵活调整。通常情况下我们需要调大它们的默认值，上面列出的 tcp_wmem 的 
min、default、max 这几组数值就是调大后的值，也是在生产环境中建议配置的值。
之所以将这几个值给调大，是因为在生产环境中遇到过 TCP 发送缓冲区太小，导致业务延迟很大的问题，这类问题也是可以使用systemtap 
之类的工具在内核里面打点来进行观察的(观察 sk_stream_wait_memory 这个事件):

```c
# sndbuf_overflow.stp
2 # Usage :
3 # $ stap sndbuf_overflow.stp
4 probe kernel.function("sk_stream_wait_memory") 5{
6 printf("%d %s TCP send buffer overflow\n", 7 pid(), execname())
8}
```

如果你可以观察到 sk_stream_wait_memory 这个事件，就意味着 TCP 发送缓冲区太小了，你需要继续去调大 wmem_max 和 
tcp_wmem:max 的值了。


## 3.2 tcp_mem 控制TCP 连接消耗的总内存
tcp_wmem 以及 wmem_max 的大小设置都是针对单个 TCP 连接的，这两个值的单位都是 Byte(字节)。系统中可能会存在非常多的
TCP 连接，如果 TCP 连接太多，就可能导致内存耗尽。因此，所有 TCP 连接消耗的总内存也有限制:

```shell
net.ipv4.tcp_mem = 8388608 12582912 16777216
```

我们通常也会把这个配置项给调大。与前两个选项不同的是，该选项中这些值的单位是 Page(页数)，也就是 4K。它也有 3 个值:
min、pressure、max。当所有 TCP 连接消 耗的内存总和达到 max 后，也会因达到限制而无法再往外发包。
因 tcp_mem 达到限制而无法发包或者产生抖动的问题，我们也是可以观测到的。为了方便地观测这类问题，Linux 内核里面预置了
静态观测点:sock_exceed_buf_limit。

观察时你只需要打开 tracepiont(需要 4.16+ 的 内核版本):

```shell
$ echo 1 > /sys/kernel/debug/tracing/events/sock/sock_exceed_buf_limit/enable
```

然后去看是否有该事件发生:

```shell
$ cat /sys/kernel/debug/tracing/trace_pipe
```

如果有日志输出(即发生了该事件)，就意味着你需要调大 tcp_mem 了，或者是需要断开一些 TCP 连接了。

## 3.3 ip_local_port_range 控制和其他服务器建立 IP 连接时本地端口(local port)的范围
TCP 层处理完数据包后，就继续往下来到了 IP 层。IP 层这里容易触发问题的地方是 net.ipv4.ip_local_port_range 这个配置选项，
它是指和其他服务器建立 IP 连接时本地端口(local port)的范围。我们在生产环境中就遇到过默认的端口范围太小，以致于无法创建新连接
的问题。所以通常情况下，我们都会扩大默认的端口范围:

```shell
net.ipv4.ip_local_port_range = 1024 65535
```

## 3.4 txqueuelen 控制qdisc 队列的长度
为了能够对 TCP/IP 数据流进行流控，Linux 内核在 IP 层实现了 qdisc(排队规则)。我们平时用到的 TC 就是基于 qdisc 的流控工具。
qdisc 的队列长度是我们用 ifconfig 来看到的 txqueuelen，我们在生产环境中也遇到过因为 txqueuelen 太小导致数据包被丢弃的情况，
这类问题可以通过下面这个命令来观察:

```shell
$ ip -s -s link ls dev eth0
...
TX: bytes packets errors dropped carrier collsns 3263284 25060 0 0 0 0
```
如果观察到 dropped 这一项不为 0，那就有可能是 txqueuelen 太小导致的。当遇到这种情况时，就需要增大该值了，比如增加 eth0 
这个网络接口的 txqueuelen:

```shell
$ ifconfig eth0 txqueuelen 2000
```

或者使用 ip 这个工具:

```shell
$ ip link set eth0 txqueuelen 2000
```

在调整了 txqueuelen 的值后，你需要持续观察是否可以缓解丢包的问题，这也便于你将它调整到一个合适的值。


## 3.5 default_qdisc 控制qdisc处理顺序
Linux 系统默认的 qdisc 为 pfifo_fast(先进先出)，通常情况下我们无需调整它。如果你想使用TCP BBR来改善 TCP 
拥塞控制的话，那就需要将它调整为 fq(fair queue, 公平队列):

```shell
net.core.default_qdisc = fq
```

经过 IP 层后，数据包再往下就会进入到网卡了，然后通过网卡发送出去。至此，你需要发送出去的数据就走完了 TCP/IP 协议栈，
然后正常地发送给对端了。


# 4 TCP 数据包的接收过程会受什么影响？





![tcp-receive-packet-process.png](images%2Ftcp-receive-packet-process.png)





## 4.1 netdev_budget 控制CPU 一次性地去批量轮询 (poll)数据包的数量
从上图可以看出，TCP 数据包的接收流程在整体上与发送流程类似，只是方向是相反的。 数据包到达网卡后，就会触发
中断(IRQ)来告诉 CPU 读取这个数据包。但是在高性能网络场景下，数据包的数量会非常大，如果每来一个数据包都要
产生一个中断，那 CPU 的处理效率就会大打折扣，所以就产生了 NAPI(New API)这种机制让 CPU 一次性地去轮询
(poll)多个数据包，以批量处理的方式来提升效率，降低网卡中断带来的性能开销。
那在 poll 的过程中，一次可以 poll 多少个呢?这个 poll 的个数可以通过 sysctl 选项来控制:

```shell
net.core.netdev_budget = 600
```

该控制选项的默认值是 300，在网络吞吐量较大的场景中，我们可以适当地增大该值，比如增大到600。增大该值可以
一次性地处理更多的数据包。但是这种调整也是有缺陷的， 因为这会导致 CPU 在这里 poll 的时间增加，如果系统
中运行的任务很多的话，其他任务的调度延迟就会增加。

## 4.2 tcp_rmem 控制TCP 接收缓冲区的大小
我们刚才提到，数据包到达网卡后会触发 CPU 去 poll 数据包，这些 poll 的数据包紧接着就会到达 IP 层去处理，
然后再达到 TCP 层，这时就会面对另外一个很容易引发问题的地方了:TCP Receive Buffer(TCP 接收缓冲区)。
与 TCP 发送缓冲区类似，TCP 接收缓冲区的大小也是受控制的。通常情况下，默认都是使用 tcp_rmem 来控制缓冲区的大小。
同样地，我们也会适当地增大这几个值的默认值，来获取更好的网络性能，调整为如下数值:

```shell
net.ipv4.tcp_rmem = 8192 87380 16777216
```

它也有 3 个字段:min、default、max。TCP 接收缓冲区大小也是在 min 和 max 之间动态调整 ，不过跟发送缓冲区
不同的是，这个动态调整是可以通过控制选项来关闭的，这个选项是 tcp_moderate_rcvbuf 。通常我们都是打开它，
这也是它的默认值:

```shell
net.ipv4.tcp_moderate_rcvbuf = 1
```

之所以接收缓冲区有选项可以控制自动调节，而发送缓冲区没有，那是因为 TCP 接收缓冲区会直接影响 TCP 拥塞控制，
进而影响到对端的发包，所以使用该控制选项可以更加灵活地控制对端的发包行为。

除了 tcp_moderate_rcvbuf 可以控制 TCP 接收缓冲区的动态调节外，也可以通过 setsockopt() 中的配置选项
SO_RCVBUF 来控制，这与 TCP 发送缓冲区是类似的。如果 应用程序设置了 SO_RCVBUF 这个标记，那么 TCP 接收
缓冲区的动态调整就是关闭，即使 tcp_moderate_rcvbuf 为 1，接收缓冲区的大小始终就为设置的 SO_RCVBUF 这个值。
也就是说，只有在 tcp_moderate_rcvbuf 为 1，并且应用程序没有通过 SO_RCVBUF 来配置缓冲区大小的情况下，
TCP接收缓冲区才会动态调节。

同样地，与 TCP 发送缓冲区类似，SO_RCVBUF 设置的最大值也不能超过 net.core.rmem_max。通常情况下，我们也需要
设置 net.core.rmem_max 的值大于等于 net.ipv4.tcp_rmem 的 max:

```shell
net.core.rmem_max = 16777216
```

我们在生产环境中也遇到过，因达到了 TCP 接收缓冲区的限制而引发的丢包问题。但是这类问题不是那么好追踪的，没有一种
很直观地追踪这种行为的方式，所以我便在我们的内核里添加了针对这种行为的统计。
为了让使用 Linux 内核的人都能很好地观察这个行为，我也把我们的实践贡献给了 Linux 内核社区，具体可以看这个 commit:
tcp: add new SNMP counter for drops when try to queue in rcv queue。使用这个 SNMP 计数，我们就可以
很方便地通过 netstat 查看，系统中是否存在因为 TCP 接收缓冲区不足而引发的丢包。
不过，该方法还是存在一些局限:如果我们想要查看是哪个 TCP 连接在丢包，那么这种方式就不行了，这个时候我们就需要去借助
其他一些更专业的 trace 工具，比如 eBPF 来达到我们的目的。

## 4.3 使用 SNMP 计数与 netstat 检查系统是否因 TCP 接收缓冲区不足而导致丢包

> 了解 TCP 接收缓冲区不足导致的丢包

TCP 是面向连接的协议，它保证数据可靠传输。每个 TCP 连接都有一个接收缓冲区，用来存放接收到但尚未被应用程序处理的数据。
如果接收缓冲区已满，而数据继续到来，TCP 协议栈将无法接收新的数据，这就可能导致丢包或发送端停止发送。

TCP 接收缓冲区不足的常见原因：
网络传输速率高于应用程序处理速度。
接收端的系统资源不足（例如内存不足）。

当接收缓冲区不足时，系统的 SNMP 计数器（如 TCPInErrs 和 TCPInCsumErrors）会增加，反映出接收端处理不当的问题。

> 使用 SNMP 计数监控 TCP 错误

SNMP 计数器可以通过 netstat 或 cat /proc/net/snmp 命令查看，它们可以提供关于系统网络协议的详细统计信息。重点关注
TCP 协议部分的以下计数器：

•TcpInErrs: 表示由于某些原因导致的 TCP 输入错误（如接收缓冲区不足）。
•TcpInCsumErrors: 检测到的 TCP 校验和错误，可能与丢包相关。

你可以使用以下命令查看这些计数器：

```shell
cat /proc/net/snmp | grep Tcp
```

或者

```shell
netstat -s | grep -i tcp
```

输出类似如下：

```shell
Tcp: RtoAlgorithm RtoMin RtoMax MaxConn ActiveOpens PassiveOpens AttemptFails EstabResets CurrEstab InSegs OutSegs RetransSegs InErrs OutRsts InCsumErrors
Tcp: 1 200 120000 -1 1443035 29354 1294 17145 237 103261482 109376062 34964 0 2415 2341
```

> 重点关注的字段解释

InErrs：表示输入错误的数量，可能由于接收缓冲区问题而引发。
InCsumErrors：TCP 校验和错误，可能与数据包丢失有关。
InSegs：系统收到的 TCP 段数。
RetransSegs：系统发出的 TCP 重传段数。如果重传段数高，通常表明网络存在丢包或网络抖动。

> 判断 TCP 接收缓冲区不足

如果你观察到 TcpInErrs 或 TcpInCsumErrors 数量增加，同时伴随着大量的 RetransSegs（TCP 重传段数增加），
这可能是由于接收缓冲区不足导致的丢包问题。此时可以进一步检查以下几个方面：

接收缓冲区大小：你可以使用 sysctl 查看和调整系统的接收缓冲区大小。例如：

```shell
sysctl net.ipv4.tcp_rmem
```
输出通常类似：

```shell
net.ipv4.tcp_rmem = 4096 87380 6291456
```

这代表最小值、默认值和最大值。适当增加最大值可以防止接收缓冲区过小导致的丢包。

TCP窗口缩放：通过查看 /proc/net/snmp 中的相关字段，如 InSegs 和 InErrs，确定窗口大小是否合适。