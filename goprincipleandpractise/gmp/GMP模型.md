
---
GMP模型
---

# 1 Goroutine与线程

## 1.1 可增长的栈

OS线程一般有固定的栈内存（通常为2MB），而goroutine的栈在其生命周期开始时只有很小的栈（典型情况下2KB），
可以按需增大和缩小，最大可达1GB。因此在Go中一次创建十万级别的goroutine也是可行的。

| 对比项 | OS线程 | Goroutine |
|--------|--------|-----------|
| 栈大小 | 固定2MB | 初始2KB，动态伸缩 |
| 创建开销 | 较大（内核态） | 极小（用户态） |
| 切换开销 | ~1-5μs（涉及内核态切换） | ~200ns（用户态切换） |
| 数量级 | 千级别 | 百万级别 |


# 2 GMP模型概述

## 2.1 演进背景：从GM到GMP

Go早期的调度器只有G和M两个组件（GM模型），所有G都放在一个全局队列中，M从全局队列取G执行。
这个设计存在严重的性能问题：

- **全局锁竞争**：每次M获取或归还G都需要加锁，高并发下锁竞争激烈
- **M之间无法高效传递G**：一个M创建的G，往往被同一个M执行，局部性差
- **系统调用导致阻塞**：M执行系统调用时无法释放资源给其他G

Go 1.1引入了P（Processor），形成了现在的GMP模型。P的核心作用是**提供本地队列，消除全局锁竞争**。

## 2.2 三个核心组件

**G（Goroutine）** — 用户态协程

g结构体包含：栈信息、当前状态、函数指针、上下文信息（PC/SP）。

G的生命周期状态：
```
_Gidle → _Grunnable → _Grunning → _Gwaiting → _Grunnable → ... → _Gdead
                           ↓
                      _Gsyscall
```

- `_Grunnable`：已就绪，等待被调度
- `_Grunning`：正在某个M上执行
- `_Gwaiting`：因channel/mutex/timer等阻塞
- `_Gsyscall`：正在执行系统调用
- `_Gdead`：执行完毕，等待回收复用

**P（Processor）** — 逻辑处理器

- 维护一个本地G队列（最多256个）
- 存储当前goroutine运行的上下文环境
- M必须绑定一个P才能执行G
- P的数量由`GOMAXPROCS`决定，默认等于CPU核心数

**M（Machine）** — 操作系统线程

- 与内核线程一一映射
- 是G实际执行的载体
- 某一时刻一个M只能运行一个G
- M的数量受`runtime/debug.SetMaxThreads()`限制，默认上限10000（注意：不是GOMAXPROCS，GOMAXPROCS限制的是P的数量）

三者的关系：**P管理着一组G，挂载在一个M上运行。**


# 3 调度机制

## 3.1 调度循环

调度器的核心是`runtime.schedule()`函数，每个M在执行完一个G后都会调用它来获取下一个G。
获取G的优先级顺序：

1. **每61次调度检查一次全局队列**（防止全局队列饥饿）
2. 从当前P的本地队列取
3. 如果本地队列为空，调用`findrunnable()`：
   - 从全局队列取（加锁）
   - 从网络轮询器（netpoller）获取就绪的G
   - 从其他P的本地队列**偷取一半**（Work Stealing）
4. 如果所有途径都找不到G，M进入休眠状态

```
schedule() → 找到G → execute(G) → G执行完毕/被抢占 → schedule() → ...
```

## 3.2 Hand-off机制

当G进入系统调用（如文件I/O）时，M也会被阻塞。为避免整个P被卡住，运行时执行hand-off：

1. G进入`_Gsyscall`状态
2. P与当前M解绑
3. P绑定到一个空闲M（或创建新M）继续执行队列中的其他G
4. 原M在系统调用返回后，尝试重新获取一个P；若没有空闲P，G被放入全局队列，M进入休眠

## 3.3 Work Stealing（工作窃取）

当一个P的本地队列为空时，会随机选择另一个P，偷取其本地队列中**一半**的G。
这保证了所有P的负载大致均衡，避免出现某些P繁忙而其他P空闲的情况。

## 3.4 Spinning线程

为了减少调度延迟，Go保持少量M处于”自旋”（spinning）状态——它们没有执行G，但也没有休眠，
而是在积极寻找可执行的G。这避免了频繁的线程唤醒开销。

自旋线程的数量有上限：最多`GOMAXPROCS`个。当自旋线程找到G后立即执行，不需要唤醒开销。


# 4 抢占式调度

## 4.1 协作式抢占（Go 1.2 ~ 1.13）

Go编译器在函数入口处插入`runtime.morestack()`检查。当sysmon检测到一个G运行超过10ms时，
会设置该G的抢占标志。G在下一次函数调用时检查到标志，主动让出CPU。

**问题**：如果G在执行一个没有函数调用的紧密循环（如`for { i++ }`），则永远不会被抢占，
可能导致其他G饥饿甚至GC无法启动。

## 4.2 异步抢占（Go 1.14+）

Go 1.14引入了基于**信号**的异步抢占机制，彻底解决了紧密循环无法抢占的问题：

1. sysmon检测到G运行超过10ms
2. 向目标M发送`SIGURG`信号
3. M的信号处理函数将当前G的执行上下文保存，并切换到调度器
4. 调度器将G重新放入队列，选择下一个G执行

这意味着即使是`for {}`这样的空循环，也能被抢占。


# 5 网络轮询器（Netpoller）

网络I/O是服务端程序最常见的阻塞操作。如果每次网络I/O都阻塞M，线程数量会迅速膨胀。
Go通过集成操作系统的I/O多路复用机制（Linux的epoll、macOS的kqueue）来解决这个问题。

**工作流程：**

1. G执行网络读写时，如果数据未就绪，G不会阻塞M
2. runtime将该socket的fd注册到netpoller（epoll/kqueue），G状态变为`_Gwaiting`
3. M释放该G，继续执行其他G
4. netpoller在后台监听fd事件，当数据就绪时，将对应的G标记为`_Grunnable`
5. 调度器在下一次调度循环中取走这些就绪的G执行

**关键优势**：网络I/O不消耗线程。无论有多少G在等待网络数据，都不需要额外的M。
这是Go能用少量线程支撑大量并发连接的根本原因。

与系统调用阻塞的区别：

| 阻塞类型 | M是否被阻塞 | 处理方式 |
|---------|------------|---------|
| 网络I/O | 否 | netpoller异步监听，M继续服务其他G |
| 文件I/O/系统调用 | 是 | hand-off：P解绑M，绑定新M |
| channel/mutex | 否 | G进入等待队列，M继续服务其他G |


# 6 sysmon监控线程

sysmon是一个特殊的M，**不需要绑定P**，独立运行，负责整个运行时的健康监控：

**核心职责：**

- **抢占长时间运行的G**：检测运行超过10ms的G，触发抢占（协作式或信号式）
- **回收syscall中的P**：如果M在系统调用中阻塞超过10μs（后逐步退避到10ms），sysmon会将P从M上解绑
- **推动网络轮询**：定期调用`netpoll()`检查是否有网络I/O就绪的G
- **触发GC**：检查是否需要强制执行垃圾回收
- **检测死锁**：所有G都在休眠且没有定时器时，报告死锁

sysmon的检查频率从20μs开始，如果长时间无事可做会逐步退避到10ms。


# 7 深入细节

## 7.1 go关键字启动协程的过程

1. **创建G**：运行时调用`runtime.newproc()`，从G的空闲池（gfree list）获取或新建一个g结构体
2. **初始化栈和上下文**：分配2KB栈空间，将函数入口和参数写入栈，状态设为`_Grunnable`
3. **放入运行队列**：优先放入当前P的本地队列；如果本地队列满（256个），则将本地队列一半的G连同新G一起转移到全局队列
4. **唤醒M**：如果有空闲P但没有spinning的M，调用`runtime.wakep()`唤醒或创建M来执行

## 7.2 阻塞G的去向

阻塞的G不在P的本地队列中（本地队列只保留`_Grunnable`状态的G）：

| 阻塞原因 | G的去向 | 唤醒机制 |
|---------|--------|---------|
| 系统调用 | 与原M绑定，状态为`_Gsyscall` | 系统调用返回后M尝试重新获取P |
| channel操作 | 挂到channel的sendq/recvq队列 | 对端操作时`goready()`唤醒 |
| mutex等待 | 挂到mutex的等待队列（sema） | `Unlock()`时唤醒 |
| 网络I/O | 注册到netpoller | epoll/kqueue事件就绪时唤醒 |
| time.Sleep/Timer | 加入timer堆 | 时间到期后由P的timer检查或sysmon触发 |

## 7.3 GOMAXPROCS

`GOMAXPROCS`决定了P的数量，即同时执行用户态Go代码的最大线程数。

```go
runtime.GOMAXPROCS(n)  // 设置P的数量
runtime.GOMAXPROCS(0)  // 查询当前P的数量（不修改）
```

- Go 1.5之前默认值为1（单核执行）
- Go 1.5及之后默认值等于CPU逻辑核心数
- 对于CPU密集型任务，`GOMAXPROCS`设为核心数即可
- 对于I/O密集型任务，适当增大`GOMAXPROCS`可能有帮助（但通常默认值已足够，因为netpoller不消耗P）


# 8 调度器可观测性

## 8.1 schedtrace

通过`GODEBUG`环境变量可以观察调度器的实时状态：

```bash
# 每1000ms打印一次调度器状态
GODEBUG=schedtrace=1000 ./your_program

# 输出示例：
# SCHED 0ms: gomaxprocs=8 idleprocs=6 threads=4 spinningthreads=1
#   idlethreads=1 runqueue=0 [0 0 0 0 0 0 0 0]
```

字段含义：
- `gomaxprocs`：P的数量
- `idleprocs`：空闲P的数量
- `threads`：M的总数
- `spinningthreads`：自旋M的数量
- `runqueue`：全局队列中的G数量
- `[0 0 ...]`：每个P本地队列中的G数量

增加`scheddetail=1`可以看到每个P和M的详细状态：

```bash
GODEBUG=schedtrace=1000,scheddetail=1 ./your_program
```

## 8.2 runtime包的诊断函数

```go
runtime.NumGoroutine()  // 当前goroutine总数
runtime.NumCPU()        // CPU核心数
runtime.GOMAXPROCS(0)   // 当前P的数量
```

当`runtime.NumGoroutine()`持续增长不下降时，通常意味着存在goroutine泄漏。


# 9 常见陷阱

## 9.1 Goroutine泄漏

启动goroutine后如果没有退出机制，会导致goroutine永远存活：

```go
// 泄漏：channel没有发送方，goroutine永远阻塞
func leak() {
    ch := make(chan int)
    go func() {
        val := <-ch  // 永远阻塞
        fmt.Println(val)
    }()
    // 函数返回，ch不再有发送方，goroutine泄漏
}
```

**预防**：始终确保goroutine有退出路径——使用`context.Context`取消、关闭channel、或设置超时。

## 9.2 GOMAXPROCS误解

`GOMAXPROCS`限制的是P（逻辑处理器）的数量，**不是**M（线程）的数量。
实际线程数可能远大于`GOMAXPROCS`，因为阻塞在系统调用中的M不占用P。

```go
// 即使GOMAXPROCS=1，如果有大量阻塞系统调用，
// 运行时仍然会创建多个M
runtime.GOMAXPROCS(1)
for i := 0; i < 100; i++ {
    go func() {
        // 阻塞系统调用会导致创建新M
        time.Sleep(time.Second)
    }()
}
```