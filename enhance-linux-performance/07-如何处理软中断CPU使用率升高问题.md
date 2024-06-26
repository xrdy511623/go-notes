
---
如何处理软中断CPU使用率升高问题
---

```shell
cat /proc/softirqs
```

这个指令显示的是系统运行以来的累计中断次数，我们需要关注的是中断次数的变化速率。

```shell
watch -d cat /proc/softirqs
```

比方说，如果我们关注到是NET_RX，也就是网络数据包接收软中断的变化速率最快，那我们就需要从网络接收的软中断着手，继续分析。
第一步应该观察系统的网络接收情况。

-n DEV 表示显示网络收发的报告，间隔3秒输出一组数据





![sar-network.png](images%2Fsar-network.png)





下面介绍一下sar的输出界面。从左到右依次是:
第一列表示报告的时间；
第二列IFACE表示网卡；
第三、四列: rxpck/s和txpck/s 分别表示每秒接收、发送的网络帧数，，也就是pps；
第五、六列：rxkb/s和txkb/s 分别表示每秒接收、发送的千字节数，也就是BPS；

如何计算网络帧的大小？
以平均统计数据为例，就是259*1024/195=1360.08字节，说明平均每个网络帧大约1.3kb。
TCP+IP头一共是40字节，如果网络帧的大小只有几十字节，就意味着出现了小包问题。

那么有没有办法知道这是什么样的网络帧，以及从哪里发送过来的呢？
使用tcpdump抓取网卡ens33上的包就可以了。

```shell
tcpdump -i ens33 -n tcp port 80
```





![tcpdump-net.png](images%2Ftcpdump-net.png)





Flags[S]表示是SYN包。
SYN Flood问题最简单的解决办法，就是从交换机或硬件防火墙中封禁掉来源IP，这样SYN包就无法发送到服务器中。
**虽然软中断类型很多，但我们生产环境下遇到的性能瓶颈大多是网络收发类型的软中断，特别是网络接收的软中断。**