
---
如何排查DNS解析问题？
---

DNS（Domain Name System），即域名系统，是互联网中最基础的一项服务，主要提供域名和 IP 地址之间映射关系的查询服务。
DNS 不仅方便了人们访问不同的互联网服务，更为很多应用提供了，动态服务发现和全局负载均衡（Global Server Load Balance，GSLB）的机制。
这样，DNS 就可以选择离用户最近的 IP 来提供服务。即使后端服务的 IP 地址发生变化，用户依然可以用相同域名来访问。


# 1 域名与DNS解析
域名我们本身都比较熟悉，由一串用点分隔开的字符组成，被用作互联网中的某一台或某一组计算机的名称，目的就是为了方便识别，互联网中
提供各种服务的主机位置。

要注意，域名是全球唯一的，需要通过专门的域名注册商才可以申请注册。为了组织全球互联网中的众多计算机，域名同样用点来分开，形成一个分层的结构。
而每个被点分割开的字符串，就构成了域名中的一个层级，并且位置越靠后，层级越高。
我们以极客时间的网站 time.geekbang.org 为例，来理解域名的含义。这个字符串中，最后面的 org 是顶级域名，中间的 geekbang 是二级域名，
而最左边的 time 则是三级域名。

如下图所示，注意点（.）是所有域名的根，也就是说所有域名都以点作为后缀，也可以理解为，在域名解析的过程中，所有域名都以点结束。





![dns-structure.png](images%2Fdns-structure.png)





通过理解这几个概念，你可以看出，域名主要是为了方便让人记住，而 IP 地址是机器间的通信的真正机制。把域名转换为 IP 地址的服务，
也就是我们开头提到的，域名解析服务（DNS），而对应的服务器就是域名服务器，网络协议则是 DNS 协议。

这里注意，DNS 协议在 TCP/IP 栈中属于应用层，不过实际传输还是基于 UDP 或者 TCP 协议（UDP 居多） ，并且域名服务器一般监听
在端口 53 上。

既然域名以分层的结构进行管理，相对应的，域名解析其实也是用递归的方式（从顶级开始，以此类推），发送给每个层级的域名服务器，直到得到解析结果。
不过不要担心，递归查询的过程并不需要你亲自操作，DNS 服务器会替你完成，你需要做的，就是预先配置一个可用的 DNS 服务器就可以了。

当然，我们知道，通常来说，每级 DNS 服务器，都会有最近解析记录的缓存。当缓存命中时，直接用缓存中的记录应答就可以了。如果缓存过期或者不存在，
才需要用刚刚提到的递归方式查询。

所以，系统管理员在配置 Linux 系统的网络时，除了需要配置 IP 地址，还需要给它配置 DNS 服务器，这样它才可以通过域名来访问外部服务。
比如，我的系统配置的就是 114.114.114.114 这个域名服务器。你可以执行下面的命令，来查询你的系统配置：

```shell
$ cat /etc/resolv.conf
nameserver 114.114.114.114
```

另外，DNS 服务通过资源记录的方式，来管理所有数据，它支持 A、CNAME、MX、NS、PTR 等多种类型的记录。比如：

A 记录，用来把域名转换成 IP 地址；
CNAME 记录，用来创建别名；
而 NS 记录，则表示该域名对应的域名服务器地址。

简单来说，当我们访问某个网址时，就需要通过 DNS 的 A 记录，查询该域名对应的 IP 地址，然后再通过该 IP 来访问 Web 服务。
比如，还是以极客时间的网站 time.geekbang.org 为例，执行下面的 nslookup 命令，就可以查询到这个域名的 A 记录，可以看到，
它的 IP 地址是 39.106.233.176：

```shell
nslookup time.geekbang.org
Server:		10.7.70.183
Address:	10.7.70.183#53

Non-authoritative answer:
Name:	time.geekbang.org
Address: 39.106.233.176
```

这里要注意，由于 114.114.114.114 并不是直接管理 time.geekbang.org 的域名服务器，所以查询结果是非权威的。使用上面的命令，
你只能得到 114.114.114.114 查询的结果。

前面还提到了，如果没有命中缓存，DNS 查询实际上是一个递归过程，那有没有方法可以知道整个递归查询的执行呢？
其实除了 nslookup，另外一个常用的 DNS 解析工具 dig ，就提供了 trace 功能，可以展示递归查询的整个过程。比如你可以执行下面
的命令，得到查询结果：

```shell
# +trace 表示开启跟踪查询
# +nodnssec 表示禁止 DNS 安全扩展
$ dig +trace +nodnssec time.geekbang.org
 
; <<>> DiG 9.11.3-1ubuntu1.3-Ubuntu <<>> +trace +nodnssec time.geekbang.org
;; global options: +cmd
.			322086	IN	NS	m.root-servers.net.
.			322086	IN	NS	a.root-servers.net.
.			322086	IN	NS	i.root-servers.net.
.			322086	IN	NS	d.root-servers.net.
.			322086	IN	NS	g.root-servers.net.
.			322086	IN	NS	l.root-servers.net.
.			322086	IN	NS	c.root-servers.net.
.			322086	IN	NS	b.root-servers.net.
.			322086	IN	NS	h.root-servers.net.
.			322086	IN	NS	e.root-servers.net.
.			322086	IN	NS	k.root-servers.net.
.			322086	IN	NS	j.root-servers.net.
.			322086	IN	NS	f.root-servers.net.
;; Received 239 bytes from 114.114.114.114#53(114.114.114.114) in 1340 ms
 
org.			172800	IN	NS	a0.org.afilias-nst.info.
org.			172800	IN	NS	a2.org.afilias-nst.info.
org.			172800	IN	NS	b0.org.afilias-nst.org.
org.			172800	IN	NS	b2.org.afilias-nst.org.
org.			172800	IN	NS	c0.org.afilias-nst.info.
org.			172800	IN	NS	d0.org.afilias-nst.org.
;; Received 448 bytes from 198.97.190.53#53(h.root-servers.net) in 708 ms
 
geekbang.org.		86400	IN	NS	dns9.hichina.com.
geekbang.org.		86400	IN	NS	dns10.hichina.com.
;; Received 96 bytes from 199.19.54.1#53(b0.org.afilias-nst.org) in 1833 ms
 
time.geekbang.org.	600	IN	A	39.106.233.176
;; Received 62 bytes from 140.205.41.16#53(dns10.hichina.com) in 4 ms
```

dig trace 的输出，主要包括四部分。

第一部分，是从 114.114.114.114 查到的一些根域名服务器（.）的 NS 记录。
第二部分，是从 NS 记录结果中选一个（h.root-servers.net），并查询顶级域名 org. 的 NS 记录。
第三部分，是从 org. 的 NS 记录中选择一个（b0.org.afilias-nst.org），并查询二级域名 geekbang.org. 的 NS 服务器。
最后一部分，就是从 geekbang.org. 的 NS 服务器（dns10.hichina.com）查询最终主机 time.geekbang.org. 的 A 记录。

这个输出里展示的各级域名的 NS 记录，其实就是各级域名服务器的地址，可以让你更清楚 DNS 解析的过程。 为了帮你更直观理解递归查询，
我把这个过程整理成了一张流程图，你可以保存下来理解。





![dns-analyze-process.png](images%2Fdns-analyze-process.png)





当然，不仅仅是发布到互联网的服务需要域名，很多时候，我们也希望能对局域网内部的主机进行域名解析（即内网域名，大多数情况下
为主机名）。Linux 也支持这种行为。
所以，你可以把主机名和 IP 地址的映射关系，写入本机的 /etc/hosts 文件中。这样，指定的主机名就可以在本地直接找到目标 IP。
比如，你可以执行下面的命令来操作：

```shell
$ cat /etc/hosts
127.0.0.1   localhost localhost.localdomain
::1         localhost6 localhost6.localdomain6
192.168.0.100 domain.com
```

或者，你还可以在内网中，搭建自定义的 DNS 服务器，专门用来解析内网中的域名。而内网 DNS 服务器，一般还会设置一个或多个
上游 DNS 服务器，用来解析外网的域名。

清楚域名与 DNS 解析的基本原理后，接下来，我就带你一起来看几个案例，实战分析 DNS 解析出现问题时，该如何定位。


# 2 案例准备
本次案例还是基于 Ubuntu 18.04，同样适用于其他的 Linux 系统。我使用的案例环境如下所示：

机器配置：2 CPU，8GB 内存。
预先安装 docker 等工具，如 apt install docker.io。

你可以先打开一个终端，SSH 登录到 Ubuntu 机器中，然后执行下面的命令，拉取案例中使用的 Docker 镜像：

```shell
$ docker pull feisky/dnsutils
Using default tag: latest
...
Status: Downloaded newer image for feisky/dnsutils:latest
```

然后，运行下面的命令，查看主机当前配置的 DNS 服务器：

```shell
$ cat /etc/resolv.conf
nameserver 114.114.114.114
```

可以看到，这台主机配置的 DNS 服务器是 114.114.114.114。
到这里，准备工作就完成了。接下来，我们正式进入操作环节。


# 3 案例分析

## 3.1  案例1：DNS 解析失败

首先，执行下面的命令，进入今天的第一个案例。如果一切正常，你将可以看到下面这个输出：

```shell
# 进入案例环境的 SHELL 终端中
$ docker run -it --rm -v $(mktemp):/etc/resolv.conf feisky/dnsutils bash
root@7e9ed6ed4974:/#
```

接着，继续在容器终端中，执行 DNS 查询命令，我们还是查询 time.geekbang.org 的 IP 地址：

```shell
/# nslookup time.geekbang.org
;; connection timed out; no servers could be reached
```

你可以发现，这个命令阻塞很久后，还是失败了，报了 connection timed out 和 no servers could be reached 错误。

看到这里，估计你的第一反应就是网络不通了，到底是不是这样呢？我们用 ping 工具检查试试。执行下面的命令，就可以测试本地到 114.114.114.114 的连通性：

```shell
/# ping -c3 114.114.114.114
PING 114.114.114.114 (114.114.114.114): 56 data bytes
64 bytes from 114.114.114.114: icmp_seq=0 ttl=56 time=31.116 ms
64 bytes from 114.114.114.114: icmp_seq=1 ttl=60 time=31.245 ms
64 bytes from 114.114.114.114: icmp_seq=2 ttl=68 time=31.128 ms
--- 114.114.114.114 ping statistics ---
3 packets transmitted, 3 packets received, 0% packet loss
round-trip min/avg/max/stddev = 31.116/31.163/31.245/0.058 ms

```

这个输出中，你可以看到网络是通的。那要怎么知道 nslookup 命令失败的原因呢？这里其实有很多方法，最简单的一种，就是开启
nslookup 的调试输出，查看查询过程中的详细步骤，排查其中是否有异常。

比如，我们可以继续在容器终端中，执行下面的命令：

```shell
/# nslookup -debug time.geekbang.org
;; Connection to 127.0.0.1#53(127.0.0.1) for time.geekbang.org failed: connection refused.
;; Connection to ::1#53(::1) for time.geekbang.org failed: address not available.
```

从这次的输出可以看到，nslookup 连接环回地址（127.0.0.1 和 ::1）的 53 端口失败。这里就有问题了，为什么会去连接环回地址，
而不是我们的先前看到的 114.114.114.114 呢？

你可能已经想到了症结所在——有可能是因为容器中没有配置 DNS 服务器。那我们就执行下面的命令确认一下：

```shell
/# cat /etc/resolv.conf
```

果然，这个命令没有任何输出，说明容器里的确没有配置 DNS 服务器。到这一步，很自然的，我们就知道了解决方法。在 /etc/resolv.conf 
文件中，配置上 DNS 服务器就可以了。

你可以执行下面的命令，在配置好 DNS 服务器后，重新执行 nslookup 命令。自然，我们现在发现，这次可以正常解析了：

```shell
/# echo "nameserver 114.114.114.114" > /etc/resolv.conf
/# nslookup time.geekbang.org
Server:		114.114.114.114
Address:	114.114.114.114#53
 
Non-authoritative answer:
Name:	time.geekbang.org
Address: 39.106.233.176
```

到这里，第一个案例就轻松解决了。最后，在终端中执行 exit 命令退出容器，Docker 就会自动清理刚才运行的容器。


## 3.2 DNS 解析不稳定
接下来，我们再来看第二个案例。执行下面的命令，启动一个新的容器，并进入它的终端中：

```shell
$ docker run -it --rm --cap-add=NET_ADMIN --dns 8.8.8.8 feisky/dnsutils bash
root@0cd3ee0c8ecb:/#
```

然后，跟上一个案例一样，还是运行 nslookup 命令，解析 time.geekbang.org 的 IP 地址。不过，这次要加一个 time 命令，
输出解析所用时间。如果一切正常，你可能会看到如下输出：

```shell
/# time nslookup time.geekbang.org
Server:		8.8.8.8
Address:	8.8.8.8#53
 
Non-authoritative answer:
Name:	time.geekbang.org
Address: 39.106.233.176
 
real	0m10.349s
user	0m0.004s
sys	0m0.0
```

可以看到，这次解析非常慢，居然用了 10 秒。如果你多次运行上面的 nslookup 命令，可能偶尔还会碰到下面这种错误：

```shell
/# time nslookup time.geekbang.org
;; connection timed out; no servers could be reached
 
real	0m15.011s
user	0m0.006s
sys	0m0.006s
```

换句话说，跟上一个案例类似，也会出现解析失败的情况。综合来看，现在 DNS 解析的结果不但比较慢，而且还会发生超时失败的情况。
这是为什么呢？碰到这种问题该怎么处理呢？

其实，根据前面的讲解，我们知道，DNS 解析，说白了就是客户端与服务器交互的过程，并且这个过程还使用了 UDP 协议。
那么，对于整个流程来说，解析结果不稳定，就有很多种可能的情况了。比方说：

DNS 服务器本身有问题，响应慢并且不稳定；
或者是，客户端到 DNS 服务器的网络延迟比较大；
再或者，DNS 请求或者响应包，在某些情况下被链路中的网络设备弄丢了。

根据上面 nslookup 的输出，你可以看到，现在客户端连接的 DNS 是 8.8.8.8，这是 Google 提供的 DNS 服务。对 Google 
我们还是比较放心的，DNS 服务器出问题的概率应该比较小。基本排除了 DNS 服务器的问题，那是不是第二种可能，本机到 DNS 服务器
的延迟比较大呢？
前面讲过，ping 可以用来测试服务器的延迟。比如，你可以运行下面的命令：

```shell
/# ping -c3 8.8.8.8
PING 8.8.8.8 (8.8.8.8): 56 data bytes
64 bytes from 8.8.8.8: icmp_seq=0 ttl=31 time=137.637 ms
64 bytes from 8.8.8.8: icmp_seq=1 ttl=31 time=144.743 ms
64 bytes from 8.8.8.8: icmp_seq=2 ttl=31 time=138.576 ms
--- 8.8.8.8 ping statistics ---
3 packets transmitted, 3 packets received, 0% packet loss
round-trip min/avg/max/stddev = 137.637/140.319/144.743/3.152 ms

```

从 ping 的输出可以看到，这里的延迟已经达到了 140ms，这也就可以解释，为什么解析这么慢了。实际上，如果你多次运行
上面的 ping 测试，还会看到偶尔出现的丢包现象。

```shell
$ ping -c3 8.8.8.8
PING 8.8.8.8 (8.8.8.8): 56 data bytes
64 bytes from 8.8.8.8: icmp_seq=0 ttl=30 time=134.032 ms
64 bytes from 8.8.8.8: icmp_seq=1 ttl=30 time=431.458 ms
--- 8.8.8.8 ping statistics ---
3 packets transmitted, 2 packets received, 33% packet loss
round-trip min/avg/max/stddev = 134.032/282.745/431.458/148.713 ms
```

这也进一步解释了，为什么 nslookup 偶尔会失败，正是网络链路中的丢包导致的。
碰到这种问题该怎么办呢？显然，既然延迟太大，那就换一个延迟更小的 DNS 服务器，比如电信提供的 114.114.114.114。
配置之前，我们可以先用 ping 测试看看，它的延迟是不是真的比 8.8.8.8 好。执行下面的命令，你就可以看到，它的延迟只有 31ms：

```shell
/# ping -c3 114.114.114.114
PING 114.114.114.114 (114.114.114.114): 56 data bytes
64 bytes from 114.114.114.114: icmp_seq=0 ttl=67 time=31.130 ms
64 bytes from 114.114.114.114: icmp_seq=1 ttl=56 time=31.302 ms
64 bytes from 114.114.114.114: icmp_seq=2 ttl=56 time=31.250 ms
--- 114.114.114.114 ping statistics ---
3 packets transmitted, 3 packets received, 0% packet loss
round-trip min/avg/max/stddev = 31.130/31.227/31.302/0.072 ms
```

这个结果表明，延迟的确小了很多。我们继续执行下面的命令，更换 DNS 服务器，然后，再次执行 nslookup 解析命令：

```shell
/# echo nameserver 114.114.114.114 > /etc/resolv.conf
/# time nslookup time.geekbang.org
Server:		114.114.114.114
Address:	114.114.114.114#53
 
Non-authoritative answer:
Name:	time.geekbang.org
Address: 39.106.233.176
 
real    0m0.064s
user    0m0.007s
sys     0m0.006s
```

你可以发现，现在只需要 64ms 就可以完成解析，比刚才的 10s 要好很多。

到这里，问题看似就解决了。不过，如果你多次运行 nslookup 命令，估计就不是每次都有好结果了。比如，在我的机器中，就经常
需要 1s 甚至更多的时间。

```shell
/# time nslookup time.geekbang.org
Server:		114.114.114.114
Address:	114.114.114.114#53
 
Non-authoritative answer:
Name:	time.geekbang.org
Address: 39.106.233.176
 
real	0m1.045s
user	0m0.007s
sys	0m0.004s
```

1s 的 DNS 解析时间还是太长了，对很多应用来说也是不可接受的。那么，该怎么解决这个问题呢？我想你一定已经想到了，那就是使用
DNS 缓存。这样，只有第一次查询时需要去 DNS 服务器请求，以后的查询，只要 DNS 记录不过期，使用缓存中的记录就可以了。

不过要注意，我们使用的主流 Linux 发行版，除了最新版本的 Ubuntu （如 18.04 或者更新版本）外，其他版本并没有自动配置 DNS 缓存。
所以，想要为系统开启 DNS 缓存，就需要你做额外的配置。比如，最简单的方法，就是使用 dnsmasq。
dnsmasq 是最常用的 DNS 缓存服务之一，还经常作为 DHCP 服务来使用。它的安装和配置都比较简单，性能也可以满足绝大多数应用程序对
DNS 缓存的需求。

我们继续在刚才的容器终端中，执行下面的命令，就可以启动 dnsmasq：

```shell
/# /etc/init.d/dnsmasq start
 * Starting DNS forwarder and DHCP server dnsmasq                    [ OK ]
```

然后，修改 /etc/resolv.conf，将 DNS 服务器改为 dnsmasq 的监听地址，这儿是 127.0.0.1。接着，重新执行多次 nslookup 命令：

```shell
/# echo nameserver 127.0.0.1 > /etc/resolv.conf
/# time nslookup time.geekbang.org
Server:		127.0.0.1
Address:	127.0.0.1#53
 
Non-authoritative answer:
Name:	time.geekbang.org
Address: 39.106.233.176
 
real	0m0.492s
user	0m0.007s
sys	0m0.006s
 
/# time nslookup time.geekbang.org
Server:		127.0.0.1
Address:	127.0.0.1#53
 
Non-authoritative answer:
Name:	time.geekbang.org
Address: 39.106.233.176
 
real	0m0.011s
user	0m0.008s
sys	0m0.003s
```

现在我们可以看到，只有第一次的解析很慢，需要 0.5s，以后的每次解析都很快，只需要 11ms。并且，后面每次 DNS 解析
需要的时间也都很稳定。

### dnsmasq的工作流程
当你启动 dnsmasq 并将 /etc/resolv.conf 中的 DNS 服务器设置为 127.0.0.1 后，工作流程如下：

> 系统发起 DNS 查询：当你访问一个域名时，例如www.example.com，系统会发起一个 DNS 查询。
> 查询被发送到 127.0.0.1：因为 /etc/resolv.conf 中的 DNS 服务器地址是 127.0.0.1，系统会将这个查询请求发送给本地的 dnsmasq。
> dnsmasq 处理查询：
  如果 dnsmasq 已经缓存了这个域名的查询结果，它会直接返回结果。
  如果没有缓存结果，它会将查询转发给上游 DNS 服务器（这些服务器是在 dnsmasq 的配置文件中指定的，如 /etc/dnsmasq.conf 或 /etc/resolv.dnsmasq.conf），然后将结果返回给系统，同时缓存这个结果。
  系统收到 DNS 解析结果：然后继续后续的网络通信过程。

总结
设置 127.0.0.1 作为 DNS 服务器地址的原因是为了让本地的 dnsmasq 处理所有的 DNS 查询，以便利用其缓存功能、减少网络延迟、以及实现
本地网络设备名称的解析等功能。虽然 127.0.0.1 是本地环回地址，但它在这种情况下被用来和本地的 dnsmasq 进程通信。


# 4 小结

DNS 是互联网中最基础的一项服务，提供了域名和 IP 地址间映射关系的查询服务。很多应用程序在最初开发时，并没考虑 DNS 解析
的问题，后续出现问题后，排查好几天才能发现，其实是 DNS 解析慢导致的。

试想，假如一个 Web 服务的接口，每次都需要 1s 时间来等待 DNS 解析，那么，无论你怎么优化应用程序的内在逻辑，对用户来说，
这个接口的响应都太慢，因为响应时间总是会大于 1 秒的。

所以，在应用程序的开发过程中，我们必须考虑到 DNS 解析可能带来的性能问题，掌握常见的优化方法。这里总结了几种常见的 DNS 优化方法。
对 DNS 解析的结果进行缓存。缓存是最有效的方法，但要注意，一旦缓存过期，还是要去 DNS 服务器重新获取新记录。不过，这对大部分应用
程序来说都是可接受的。
对 DNS 解析的结果进行预取。这是浏览器等 Web 应用中最常用的方法，也就是说，不等用户点击页面上的超链接，浏览器就会在后台自动解析
域名，并把结果缓存起来。
使用 HTTPDNS 取代常规的 DNS 解析。这是很多移动应用会选择的方法，特别是如今域名劫持普遍存在，使用 HTTP 协议绕过链路中的 DNS 
服务器，就可以避免域名劫持的问题。
基于 DNS 的全局负载均衡（GSLB）。这不仅为服务提供了负载均衡和高可用的功能，还可以根据用户的位置，返回距离最近的 IP 地址。


## 域名劫持
域名劫持（Domain Hijacking）是指未经授权的情况下，恶意攻击者通过控制域名解析、篡改域名注册信息或劫持域名解析服务等手段，
将用户对某个合法域名的访问请求重定向到攻击者控制的IP地址或恶意网站。域名劫持是一种网络攻击，通常用于窃取敏感信息、
传播恶意软件、实施网络钓鱼或其他恶意行为。

**域名劫持的常见类型**

	1.	DNS劫持（DNS Hijacking）：
		描述：攻击者通过篡改DNS服务器或用户本地的DNS设置，将对某个域名的DNS查询结果指向错误的IP地址。
		实现方式：
		攻击者控制了用户的本地DNS服务器，通过篡改其解析记录，使用户访问合法域名时被重定向到恶意网站。
		攻击者通过恶意软件或病毒感染用户的设备，修改其/etc/hosts文件（在Windows、macOS和Linux中均存在）或操作系统的DNS设置，将特定域名解析到恶意IP地址。
		影响：用户在浏览器中输入合法域名后，可能会被重定向到与合法网站完全无关的恶意网站，导致信息泄露或遭遇其他攻击。
	2.	域名注册信息劫持：
		描述：攻击者通过篡改域名注册信息，获得对域名的控制权，进而修改域名的DNS解析记录。
		实现方式：
		攻击者通过网络钓鱼或其他手段获取域名注册商的账户信息，并使用这些信息登录域名管理系统，篡改域名的注册信息和DNS设置。
		有时，域名注册商的安全漏洞或管理不当也可能被攻击者利用，直接篡改域名信息。
		影响：攻击者可以将该域名的所有流量引导至他们控制的服务器，导致大量用户受到影响，甚至可能丧失对域名的控制权。
	3.	缓存投毒（Cache Poisoning）：
		描述：攻击者通过向DNS服务器注入虚假的DNS解析结果，污染其缓存，从而使DNS服务器向用户返回错误的解析结果。
		实现方式：
		攻击者向DNS服务器发送大量伪造的DNS应答，试图用虚假信息覆盖正确的解析记录。
		当用户查询某个域名时，DNS服务器会返回错误的IP地址，使用户被重定向到恶意网站。
		影响：大量用户在一段时间内都会被错误的DNS解析结果误导，访问到恶意网站。

**域名劫持的影响**

	•	用户信息泄露：通过域名劫持，攻击者可以将用户引导到钓鱼网站，窃取用户名、密码、银行卡信息等敏感数据。
	•	传播恶意软件：攻击者可以通过劫持的域名传播恶意软件，感染用户设备，进一步扩大攻击范围。
	•	品牌信誉受损：企业的域名被劫持后，可能导致客户对品牌失去信任，进而对企业的声誉和业务造成重大损失。
	•	网络钓鱼：劫持合法域名进行网络钓鱼，用户难以察觉欺骗行为，可能导致大规模的用户信息被盗。

**如何防范域名劫持**

	1.	使用DNSSEC（DNS Security Extensions）：
	    DNSSEC为DNS查询提供了数据完整性和真实性验证，能有效防止DNS缓存投毒和域名劫持。
	2.	启用双因素认证（2FA）：
	    在域名注册商账户上启用双因素认证，可以防止攻击者通过盗取登录凭据获得对域名的控制权。
	3.	使用安全的域名注册商：
	    选择提供强大安全措施的域名注册商，确保域名注册信息的安全性。
	4.	定期检查和更新DNS设置：
	    定期检查域名的DNS设置和注册信息，确保没有未经授权的修改。
	5.	监控域名解析行为：
	    使用监控工具监视域名解析行为，一旦发现异常立即采取行动。
	6.	教育用户防范网络钓鱼：
	    提高用户的安全意识，防范通过网络钓鱼获取敏感信息的攻击。


## HTTPDNS

HTTPDNS 是一种通过 HTTP 协议直接向特定的 DNS 服务器请求域名解析结果的技术，绕过了传统的基于 UDP 协议的 DNS 解析。
这种方法特别适合在移动应用中使用，可以有效防止域名劫持等安全问题。

**HTTPDNS 的工作原理**

****传统DNS解析****
用户设备向默认的 DNS 服务器（通常是 ISP 提供的 DNS）发起域名解析请求。
DNS 服务器通过一系列递归查询，将域名转换为对应的 IP 地址，并将结果返回给用户设备。
这种方法依赖于网络中间的 DNS 服务器，因此存在被篡改或劫持的风险。

**HTTPDNS 解析**
用户设备通过 HTTP 或 HTTPS 协议，直接向指定的 HTTPDNS 服务器发送域名解析请求。
HTTPDNS 服务器接收到请求后，查询域名的 IP 地址，并将结果通过 HTTP/HTTPS 响应返回给用户设备。
由于 HTTPDNS 使用的是基于 TCP 的 HTTP 协议，且请求目标是一个固定的、受信任的 DNS 服务器，整个过程不会经过普通的
DNS 服务器，减少了被劫持或篡改的风险。

**HTTPDNS 的优点**
防止域名劫持：
    传统的 DNS 请求可能在网络传输中被拦截或篡改，导致域名劫持。HTTPDNS 通过使用 HTTP/HTTPS 协议，可以避免这些攻击。
提高解析的准确性和稳定性：
   HTTPDNS 直接访问权威的 DNS 服务器，避免了本地 ISP DNS 服务器的不稳定性，确保解析结果的准确性。
绕过 DNS 污染：
    在某些地区，DNS 污染（DNS 污染是一种通过篡改 DNS 解析结果来进行审查的技术）可能导致某些域名无法正常解析。
    HTTPDNS 可以绕过这些本地 DNS 服务器，获取正确的解析结果。
支持移动网络的优化： 
   在移动网络环境中，HTTPDNS 可以减少运营商 DNS 的不稳定性，避免因网络切换导致的 DNS 缓存问题，提升用户体验。

**HTTPDNS 的实现方式**
通过公共 HTTPDNS 服务： 
   一些云服务提供商（如阿里云、腾讯云等）提供公共的 HTTPDNS 服务，应用开发者可以通过这些服务实现域名解析。例如，阿里云
   的 HTTPDNS 服务允许开发者通过 HTTP/HTTPS 请求获取域名解析结果，并提供 SDK 方便集成。

自建 HTTPDNS 服务： 
    大型企业或对安全要求高的应用可以选择自建 HTTPDNS 服务器，通过自有的权威 DNS 服务器提供解析服务，并通过 HTTP 接口对外提供服务。

**HTTPDNS 的应用场景**
    移动应用：
          •	在移动网络中，HTTPDNS 可以避免因网络切换和运营商 DNS 不稳定导致的解析问题，提供更加稳定的解析服务。
    跨境电商：
          •	跨境电商平台经常需要在不同国家和地区部署服务。HTTPDNS 可以避免由于不同国家的 DNS 污染导致的解析问题，确保用户能正常访问。
    对抗 DNS 劫持：
          •	对于需要防止 DNS 劫持的应用场景（如金融服务、敏感信息传输等），HTTPDNS 是一种有效的防护措施。
    内容分发网络（CDN）：
          •	CDN 服务商可以通过 HTTPDNS 提供更精确的解析服务，将用户引导至离其最近的节点，提升访问速度。

**HTTPDNS 的局限性**
    依赖 HTTPDNS 服务器的可靠性：
          •	HTTPDNS 服务的可用性和稳定性依赖于其服务器的可靠性，如果 HTTPDNS 服务器出现问题，解析服务将会中断。
    增加网络延迟：
          •	因为 HTTPDNS 基于 HTTP/HTTPS 协议，增加了 TCP 握手的过程，可能会引入一些额外的网络延迟。
    负载均衡和 CDN 的问题：
          •	某些基于 DNS 的负载均衡或 CDN 分发策略可能无法通过 HTTPDNS 正确实现，需要对这些应用场景进行额外优化。

**HTTPDNS 的未来发展**

随着对网络安全要求的提高和 DNS 劫持问题的普遍存在，HTTPDNS 的应用将越来越广泛。未来，可能会有更多的企业和应用选择采用
HTTPDNS 来提升域名解析的安全性和稳定性，同时，HTTPDNS 技术本身也会不断演进，以更好地适应复杂的网络环境和多样化的应用需求。