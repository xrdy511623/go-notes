
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





使用top工具查看占用CPU资源最多的函数，定位到*Tiger的Eat函数

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

# 4 go tool trace —— pprof 的互补工具

pprof 告诉你"哪里慢"，trace 告诉你"为什么慢"。两者互补，缺一不可。

## 4.1 采集与查看

```bash
# 方式一：通过 HTTP 端点采集（适合运行中的服务）
curl -o trace.out "http://localhost:6060/debug/pprof/trace?seconds=5"
go tool trace trace.out

# 方式二：在代码中手动采集
# 见 performance/trace_demo_test.go
```

## 4.2 trace 能看到而 pprof 看不到的

| 维度 | pprof | trace |
|------|-------|-------|
| CPU 耗时 | 函数级别采样统计 | 时间线上每个事件的精确时间戳 |
| GC 影响 | 只能看到 GC 函数的 CPU 占比 | 能看到 STW 暂停在时间线上的精确位置和持续时间 |
| 调度延迟 | 看不到 | 能看到 goroutine 等待被调度的时间 |
| 系统调用 | 看不到 | 能看到每次 syscall 的阻塞时间 |
| 并行度 | 看不到 | 能看到每个 P 上 goroutine 的执行时间线 |

## 4.3 trace 视图详解

打开 `go tool trace` 后，浏览器会显示以下视图：

**Goroutines analysis**
按 goroutine 分组，展示每个 goroutine 的生命周期：创建、运行、等待、阻塞、GC 辅助。

**Network/Sync/Syscall blocking profile**
分别展示网络 I/O、同步原语、系统调用造成的阻塞耗时排行。

**Scheduler latency profile**
展示 goroutine 从"就绪"到"运行"的调度延迟分布，可以发现 P 不够用或 GOMAXPROCS 设置不合理的问题。

**User-defined tasks & regions**
通过 `runtime/trace` 包自定义标记：

```go
import "runtime/trace"

// 标记一个任务（跨多个 goroutine）
ctx, task := trace.NewTask(ctx, "processOrder")
defer task.End()

// 标记一个代码区间
trace.WithRegion(ctx, "validateInput", func() {
    // ...
})
```

自定义标记在 trace 视图中会高亮显示，便于在海量事件中快速定位关注的业务逻辑。

## 4.4 实际使用场景

**场景一：GC 导致的延迟毛刺**
pprof 只能告诉你 GC 占了多少 CPU，trace 能精确告诉你"某次请求在第 3ms 处被 STW 暂停了 200μs"。

**场景二：goroutine 调度饥饿**
某些 goroutine 长时间得不到调度（被计算密集的 goroutine 抢占），pprof 无法发现，trace 能清晰展示。

**场景三：并行度不足**
程序有 8 个 P，但 trace 显示大部分时间只有 2 个 P 在工作，说明并行化不够或存在锁瓶颈。

完整示例见 `performance/trace_demo_test.go`。

# 5 pprof 进阶用法

## 5.1 对比分析（-base / -diff_base）

在生产环境中，对比优化前后的 profile 是验证效果的最可靠方式：

```bash
# 采集优化前的 profile
curl -o before.prof "http://localhost:6060/debug/pprof/heap"

# ... 部署优化后的代码 ...

# 采集优化后的 profile
curl -o after.prof "http://localhost:6060/debug/pprof/heap"

# 对比分析：只显示差异部分
go tool pprof -base before.prof after.prof
```

在交互模式下，`top` 会显示两个 profile 之间的增量：
- 正值表示优化后增加（可能是退化）
- 负值表示优化后减少（优化有效）

```bash
# 也可以直接在 web 界面对比
go tool pprof -http=:8089 -diff_base before.prof after.prof
```

## 5.2 Benchmark + pprof 联动

不需要启动 HTTP 服务，直接从 benchmark 生成 profile 文件：

```bash
# 同时生成 CPU 和内存 profile
go test -bench=BenchmarkFoo -cpuprofile=cpu.prof -memprofile=mem.prof ./...

# 分析 CPU profile
go tool pprof cpu.prof

# 分析内存 profile
go tool pprof -alloc_objects mem.prof
```

这种方式适合对单个函数进行精确的性能分析，无需启动完整的服务。

**进阶：benchmark + trace 联动**
```bash
go test -bench=BenchmarkFoo -trace=trace.out ./...
go tool trace trace.out
```

完整示例见 `performance/benchmark_pprof_test.go`。

## 5.3 逃逸分析与 pprof 联动

当 pprof 的 heap profile 发现某函数 alloc 过多时，用逃逸分析定位原因：

```bash
# 第一步：pprof 发现 Foo() 分配了大量内存
go tool pprof -alloc_objects http://localhost:6060/debug/pprof/heap
(pprof) top
(pprof) list Foo  # 定位到具体代码行

# 第二步：逃逸分析确认哪些变量逃逸到堆上
go build -gcflags='-m' ./...           # 一级输出
go build -gcflags='-m -m' ./...        # 二级详细输出，包含逃逸原因

# 第三步：根据逃逸原因优化
# 常见优化：返回值类型而非指针、避免 interface{} 装箱、预分配切片
```

## 5.4 火焰图高级用法

前面 2.2 节展示了 goroutine 火焰图，这里补充更多视图模式：

**Graph 视图（调用关系图）**
- 节点大小 = 函数自身耗时（flat）
- 边的粗细 = 调用路径上的耗时（cum）
- 红色/大节点 = 热点函数

**Flame Graph（火焰图）**
- 宽度 = 采样命中次数（CPU profile 中就是 CPU 时间占比）
- 从下到上 = 调用栈（底部是入口函数）
- 可点击放大某个函数的子调用

**Icicle Graph（倒置火焰图）**
- 与火焰图相反，从上到下表示调用栈
- 适合快速发现"被谁调用最多"

**Source 视图**
- 逐行标注 CPU 占比或内存分配量
- 等效于 `list` 命令，但以网页形式展示

```bash
# 直接在浏览器中查看所有视图
go tool pprof -http=:8089 cpu.prof
# 浏览器中可在 VIEW 菜单切换：Top / Graph / Flame Graph / Source
```

## 5.5 runtime/pprof 手动采集

在非 HTTP 服务的场景（如 CLI 工具、批处理任务），使用 `runtime/pprof` 手动采集：

```go
import "runtime/pprof"

// CPU profile
f, _ := os.Create("cpu.prof")
pprof.StartCPUProfile(f)
defer pprof.StopCPUProfile()

// Heap profile（在需要的时刻快照）
f2, _ := os.Create("mem.prof")
runtime.GC()  // 先触发 GC，确保数据准确
pprof.WriteHeapProfile(f2)
f2.Close()
```

## 5.6 自定义采样率

```go
// CPU 采样率（默认 100Hz，一般不需要改）
runtime.SetCPUProfileRate(200)  // 提高到 200Hz，采样更精确但开销更大

// Heap 采样率（默认 512KB）
runtime.MemProfileRate = 1  // 每次分配都记录（极其精确但开销很大）
runtime.MemProfileRate = 512 * 1024  // 默认值

// Mutex 采样率
runtime.SetMutexProfileFraction(5)  // 每 5 次锁操作记录 1 次

// Block 采样率
runtime.SetBlockProfileRate(1000)  // 阻塞超过 1000ns 才记录
```

注意：生产环境中不建议将采样率设置为 1，开销过大。建议仅在排查问题时临时调高。

# 6 内存泄漏实战排查

## 6.1 排查模式

内存泄漏的特征：`inuse_space` 随时间持续增长，即使负载稳定。

```bash
# 时序对比法：间隔采集两次 heap profile
curl -o heap1.prof "http://localhost:6060/debug/pprof/heap"
sleep 60
curl -o heap2.prof "http://localhost:6060/debug/pprof/heap"

# 对比两次 inuse_space，增量部分就是泄漏嫌疑
go tool pprof -inuse_space -base heap1.prof heap2.prof
(pprof) top  # 增长最多的函数就是泄漏嫌疑
```

## 6.2 常见泄漏场景

**goroutine 泄漏**（最常见）
```go
// 错误：channel 永远没有写入者，goroutine 永远阻塞
go func() {
    val := <-ch  // 永远阻塞，goroutine 无法退出
    process(val)
}()
```

排查方式：
```bash
# goroutine profile 中数量持续增长
go tool pprof "http://localhost:6060/debug/goroutine"
(pprof) top
# 如果某个创建点的 goroutine 数持续增长，就是泄漏
```

**time.After 泄漏**
```go
// 错误：循环中使用 time.After，每次迭代创建新 timer
for {
    select {
    case <-ch:
        // ...
    case <-time.After(time.Second):  // 每次循环都创建新 timer！
        // ...
    }
}

// 正确：复用 timer
timer := time.NewTimer(time.Second)
for {
    select {
    case <-ch:
        if !timer.Stop() {
            <-timer.C
        }
        timer.Reset(time.Second)
    case <-timer.C:
        timer.Reset(time.Second)
    }
}
```

**slice 底层数组未释放**
```go
// 错误：截取大 slice 的一小部分，底层大数组无法被 GC
func getHeader(data []byte) []byte {
    return data[:10]  // 持有整个底层数组的引用
}

// 正确：拷贝需要的部分
func getHeader(data []byte) []byte {
    header := make([]byte, 10)
    copy(header, data[:10])
    return header
}
```

## 6.3 自动化监控

在生产环境中，建议定期采集 profile 并持久化，便于事后分析：

```go
import "runtime/pprof"

// 每 5 分钟采集一次 heap profile
go func() {
    for range time.Tick(5 * time.Minute) {
        f, err := os.Create(fmt.Sprintf("heap-%d.prof", time.Now().Unix()))
        if err != nil {
            log.Printf("create heap profile: %v", err)
            continue
        }
        runtime.GC()
        if err := pprof.WriteHeapProfile(f); err != nil {
            log.Printf("write heap profile: %v", err)
        }
        f.Close()
    }
}()
```

对于更成熟的方案，可以使用持续性能分析平台（Continuous Profiling），如 Pyroscope、Parca 或
Google Cloud Profiler，它们能自动采集、存储和可视化 profile 数据，支持跨时间维度的对比分析。

# 7 pprof 实战 checklist

日常性能排查时，按照以下顺序逐项检查：

```
1. 浏览器打开 http://localhost:6060/debug/pprof/
   └── 快速总览各项指标

2. CPU 分析
   go tool pprof "http://localhost:6060/debug/pprof/profile?seconds=30"
   └── top → list 热点函数 → 优化

3. 内存分析
   go tool pprof -inuse_space "http://localhost:6060/debug/pprof/heap"
   └── top → 检查是否有异常大的 inuse
   go tool pprof -alloc_objects "http://localhost:6060/debug/pprof/heap"
   └── top → 找出分配最频繁的函数 → 逃逸分析确认原因

4. Goroutine 分析
   go tool pprof "http://localhost:6060/debug/pprof/goroutine"
   └── top → 数量是否异常 → 是否有泄漏

5. 锁和阻塞分析
   go tool pprof "http://localhost:6060/debug/pprof/mutex"
   go tool pprof "http://localhost:6060/debug/pprof/block"
   └── top → 是否有不合理的锁竞争或阻塞

6. Trace 分析（如果以上无法定位问题）
   curl -o trace.out "http://localhost:6060/debug/pprof/trace?seconds=5"
   go tool trace trace.out
   └── 调度延迟 → GC 影响 → 并行度
```