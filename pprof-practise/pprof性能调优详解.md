
---
pprof性能调优详解
---

1 性能调优原则
要依靠数据而不是猜测
要定位最大瓶颈而不是细枝末节
不要过早优化
不要过度优化

2 性能分析工具 pprof
说明
希望知道应用在什么地方耗费了多少CPU、Memory
pprof是用于可视化和分析性能数据的工具

2.1> pprof-功能简介
![pprof-brief.png](images%2Fpprof-brief.png)

2.2> pprof-性能排查实战

浏览器查看指标
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

接下来，使用list工具根据指定的正则表达式查找代码行
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