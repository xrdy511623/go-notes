
---
如何理解内存中的Buffer和Cache?
---

Buffer是对磁盘数据的缓存，而Cache是对文件数据的缓存，它们既会用在读请求中，也会用在写请求中。
从写的角度来说，不仅可以优化磁盘和文件的写入，对应用程序也有好处，应用程序可以在数据真正落盘前，就返回去做其他工作。

从读的角度来说，即可以加速读取那些需要频繁访问的数据，也降低了频繁I/O对磁盘的压力。

```shell
# 通过读取随机设备，生成一个500MB大小的文件，即写文件
dd if=/dev/urandom of=/tmp/file bs=1M count=500

# 首先清理缓存
echo 3 > /proc/sys/vm/drop_caches
# 然后运行dd命令向磁盘分区/dev/sda1写入2GB数据
dd if=/dev/urandom of=/dev/sda1 bs=1M count=2048

# 首先清理缓存
echo 3 > /proc/sys/vm/drop_caches
# 然后运行dd命令读取文件数据
dd if=/tmp/file of=/dev/null

# 首先清理缓存
echo 3 > /proc/sys/vm/drop_caches
# 然后运行dd命令从磁盘分区/dev/sda1读取数据，写入空设备
dd if=/dev/sda1 of=/dev/null bs=1M count=1024
```

可以通过vmstat 1 命令查看文件或磁盘读写前后Buffer 和Cache的变化情况。





![vmstat-1.png](images%2Fvmstat-1.png)





上面的bi和bo分别表示块设备读取和写入的大小，单位为块/秒。由于Linux中块的大小是1KB，所以这个单位就等价于KB/s。