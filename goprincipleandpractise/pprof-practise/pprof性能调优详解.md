
---
pprof性能调优详解
---

# 1 性能调优原则
要依靠数据而不是猜测;
要定位最大瓶颈而不是细枝末节;
不要过早优化!
不要过度优化!

# 2 性能分析工具-pprof
说明
希望知道应用在什么地方耗费了多少CPU、Memory
pprof是用于可视化和分析性能数据的工具

## 2.1 pprof-功能简介





![pprof-brief.png](images%2Fpprof-brief.png)





## 2.2 pprof-性能排查实战

> 浏览器查看指标





![overview-initial.png](images%2Foverview-initial.png)





> CPU
```shell
go tool pprof "http://localhost:6060/debug/profile?seconds=10"
```

profile表示采样的是CPU指标，seconds=10代表采样时长为10s。





![cpu-shell.png](images%2Fcpu-shell.png)





使用top工具查看暂用CPU资源最多的函数，定位到*Tiger的Eat函数

下面对top工具的输出做一简要说明:
flat 当前函数本身的执行耗时
flat% flat占CPU总时间的比例
sum% 上面每一行的flat%总和，从上到下累加flat%
cum  当前函数本身加上其调用函数的总耗时
cum% cum占CPU总时间的比例

什么情况下flat=cum? 什么情况下flat=0?
显然，当前函数没有调用其他函数时，flat=cum;
当前函数只有其他函数的调用时，flat=0

接下来，使用list工具根据指定的正则表达式查找代码行。





![list-search-problem.png](images%2Flist-search-problem.png)






```golang
loop := 10000000000
for i := 0; i < loop; i++ {
    // do nothing
}
```
可以看到这个函数里，进行了100亿次的空循环，占用了大量的CPU时间，当我们把这部分代码注释后，
CPU的性能问题也就解决了。





![cpu-solve.png](images%2Fcpu-solve.png)





> heap 堆内存

除了命令行工具外，我们也可以使用web工具对资源占用进行可视化分析

首先需要安装graphviz(mac)
```shell
brew install graphviz
```

然后将采样的资源数据展示到指定网页

```shell
go tool pprof -http=:8089 "http://localhost:6060/debug/heap"
```





![before-heap.png](images%2Fbefore-heap.png)





通过top模式定位到造成堆内存暴涨的问题代码(Mouse.Steal()以及Mouse.Pee())
我们将问题代码注释后，堆内存的问题得以解决。





![solve-heap-one.png](images%2Fsolve-heap-one.png)





![solve-heap-two.png](images%2Fsolve-heap-two.png)





内存相关指标说明





![mem-stats.png](images%2Fmem-stats.png)





alloc_objects 程序累计申请的对象数；
alloc_space   程序累计申请的内存大小；
inuse_objects 程序当前持有的对象数；
inuse_space   程序当前占用的内存大小

如果要查看这些内存指标数据，可以这样操作：

```shell
go tool pprof -seconds=30 "http://localhost:6060/debug/allocs"
```
然后输入o查看sample_index的可选值。


比如你想看alloc_space，也就是程序累计申请的内存大小，那么可以这样：
```shell
sample_index=alloc_space
top 10
```

> 协程(goroutine)

接下来我们排查程序的协程问题，通过下面的指令查看当前的协程数据

```shell
go tool pprof -http=:8089 "http://localhost:6060/debug/goroutine"
```





![before-goroutine.png](images%2Fbefore-goroutine.png)





可以看到协程数多的代码是Wolf.Drink()，我们将其注释后协程暴涨的问题得以解决。





![solve-goroutine.png](images%2Fsolve-goroutine.png)





![goroutine-flame.png](images%2Fgoroutine-flame.png)





选择上面的火焰图(flame)模式，可以更直观的定位到问题函数。
从上到下表示调用顺序；
每一块代表一个函数，越长代表占用CPU的时间更长；
火焰图是动态的，支持点击块进行分析。





![after-goroutine.png](images%2Fafter-goroutine.png)





此时，goroutine(协程)的数量由最开始的52个下降到5个。

> mutex

下面，我们来尝试排查锁的性能问题。
```shell
go tool pprof -http=:8089 "http://localhost:6060/debug/mutex"
```





![before-lock.png](images%2Fbefore-lock.png)





通过top模式我们很容易定位到问题代码是Wolf.Howl()函数，将其注释后互斥锁的问题得以解决。





![solve-lock.png](images%2Fsolve-lock.png)





![after-lock.png](images%2Fafter-lock.png)






> block

最后，我们来排查程序的block(阻塞)问题。
```shell
go tool pprof -http=:8089 "http://localhost:6060/debug/block"
```

同样的，我们通过top模式可以地轻松定位到问题代码是Cat.Pee()函数





![solve-block.png](images%2Fsolve-block.png)





在我们将问题代码注释后，阻塞的问题得以解决。





![after-block.png](images%2Fafter-block.png)





等等，我们在总览里看到程序的block阻塞有两处，为什么这里只显示了一处？





![block-explain.png](images%2Fblock-explain.png)





原因是另一处阻塞的时间太短(<=0.6s)秒的不予展示。
那另一处阻塞到底是什么情况呢？





![another-block.png](images%2Fanother-block.png)





可以看到是正常的sync.Wait操作。

所有问题解决后，我们再来看程序的主要性能指标数据:






![after-overview.png](images%2Fafter-overview.png)





# 3 性能分析工具-pprof采样过程和原理

> CPU

采样对象: 函数调用和它们占用的时间
采样率: 100次/秒，固定值
采样时间: 从手动启动和手动结束

开始采样 ----> 设定信号处理函数 ----> 开启定时器
停止采样 ----> 取消信号处理函数 ----> 关闭定时器





![cpu-record.png](images%2Fcpu-record.png)





操作系统每10ms向进程发送一次SIGPROF信号;
进程每次接收到SIGPROF信号会记录调用堆栈；
写缓冲:每100ms读取已经记录的调用堆栈并写入输出流。

> Heap 堆内存

采样程序通过内存分配器在堆上分配和释放的内存，记录分配和释放的内存大小和数量；
采样率: 每分配512KB记录一次，可在运行开头修改，1为每次分配均记录；
采样时间: 从程序运行开始到采样时；
采样指标: alloc_space, alloc_objects, inuse_space, inuse_objects。
计算方式: inuse = alloc - free
注意: Heap采样只记录大小，不记录类型信息。

> Goroutine协程 & ThreadCreate 线程创建

Goroutine
记录所有用户发起且在运行中的goroutine(即入口非runtime开头的)
runtime.main的调用栈信息。

ThreadCreate
记录程序创建的所有系统线程信息

Goroutine  stop the world ---> 遍历allg切片 ---> 输出创建g的堆栈 ---> start the world
ThreadCreate  stop the world ---> 遍历allm链表 ---> 输出创建m的堆栈 ---> start the world

> Block-阻塞 & Mutex-锁

阻塞操作：
采样阻塞操作的次数和耗时
采样率: 阻塞耗时超过阈值的才会被记录，1为每次阻塞均记录。

锁竞争
采样争抢锁的次数和耗时
采样率:只记录固定比例的锁操作，1为每次加锁均记录。