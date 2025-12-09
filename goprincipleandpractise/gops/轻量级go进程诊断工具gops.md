
---
轻量级go进程诊断工具gops
---

# 1 介绍

Go 语言的并发模型虽然强大，但也引入了新的问题类型，如死锁、活锁、goroutine 泄漏等。当这些问题发生时，应用可能表现为
失去响应、性能急剧下降或资源耗尽。诊断这类问题的关键在于，能够洞察大量 goroutine 的当前状态和它们之间的交互。

下面，我们介绍一个轻量级的进程诊断工具 gops，它可以快速获取运行中 Go 进程的 goroutine 堆栈和运行时统计信息。

gops（由 Google 开发，项目地址是 github.com/google/gops）是一个非常实用的命令行工具，用于列出当前系统上正在运行的 Go 进程，
并对它们进行一些基本的诊断操作。它的一个巨大优势是通常无需修改目标 Go 程序或重启它就能获取信息。不过，这需要你的目标 Go 程序
内像下面代码一样嵌入了 gops 的 Agent：


```go
package main

import (
    "log"
    "time"

    "github.com/google/gops/agent"
)

func main() {
    if err := agent.Listen(agent.Options{}); err != nil {
        log.Fatal(err)
    }
    // ... ...
}
```


gops的 Agent（在目标 Go 程序启动时会自动运行一小段代码）会在一个特定的位置（例如，Unix Domain Socket 或特定
TCP 端口，取决于操作系统和配置）监听来自 gops 命令行工具的连接和指令。当 gops 命令行工具执行如 gops stack 时，
它会连接到目标 Go 进程的 Agent，Agent 随后会调用 Go 运行时内部的函数 （例如，与 runtime.Stack 或 runtime/pprof 
包中获取 profile 数据相关的函数）来收集所需的信息，并将结果返回给 gops 命令行工具显示。

这种机制与 net/http/pprof 包的工作方式有相似之处，后者也是通过 HTTP 服务 暴露运行时 profile 数据的接口。实际上，
gops 提供的某些功能（如获取 CPU/Heap profile、trace 数据）底层就是触发了与 net/http/pprof 端点类似的运行时
数据收集逻辑。相比于 net/http/pprof 端点，gops 提供了一种体验更好的替代途径来获取类似的诊断信息，特别是
goroutine 堆栈、运行时统计和基本的 profile 数据。对于更深入的、可交互的 profile 分析（如生成火焰图），
net/http/pprof 的 Web 界面或 go tool pprof 仍然是首选。

gops 也支持连接到远程的 Go 应用上，前提是 Go 应用在调用 agent.Listen 时将参数 agent.Options 中的
Addr(host:port) 设置为对外部开放的 ip 和 port，不过出于安全考虑，是否这么做需要根据实际情况做出权衡。


在目标机器上，通过下面命令安装 gops 命令行工具后便可以对目标 Go 程序进行调试诊断了：


```shell
go install github.com/google/gops@latest
```


# 2 使用案例

下面我们结合一个示例程序来展示一下 gops 的主要用法以及如何基于 gops 对目标程序的并发问题进行调查和诊断。
示例的源码如下：

```go

package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/gops/agent"
)

func main() {
	if err := agent.Listen(agent.Options{}); err != nil {
		panic(err)
	}

	fmt.Printf("My PID is: %d. Waiting for deadlock...\n", os.Getpid())
	var mu1, mu2 sync.Mutex

	var wg sync.WaitGroup
	wg.Add(2)

	go func() { // Goroutine 1
		defer wg.Done()
		mu1.Lock()
		fmt.Println("G1: mu1 locked")
		time.Sleep(100 * time.Millisecond) // Give G2 time to acquire mu2
		fmt.Println("G1: Attempting to lock mu2...")
		mu2.Lock() // Will block here waiting for G2
		fmt.Println("G1: mu2 locked (should not happen in deadlock)")
		mu2.Unlock()
		mu1.Unlock()
	}()

	go func() { // Goroutine 2
		defer wg.Done()
		mu2.Lock()
		fmt.Println("G2: mu2 locked")
		time.Sleep(100 * time.Millisecond) // Give G1 time to acquire mu1
		fmt.Println("G2: Attempting to lock mu1...")
		mu1.Lock() // Will block here waiting for G1
		fmt.Println("G2: mu1 locked (should not happen in deadlock)")
		mu1.Unlock()
		mu2.Unlock()
	}()

	fmt.Println("Setup complete. Run 'gops stack <PID>' from another terminal, then send SIGINT (Ctrl+C) to stop.")
	var done = make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	for {
		select {
		case <-done:
			fmt.Println("Program close normally")
			return
		default:
			time.Sleep(5 * time.Second)
		}
	}
}
```


这是一个用两个 goroutine 展示经典的 AB-BA 死锁模式的示例。我们先把它运行起来：

```shell

$go build
$./gops_deadlock
My PID is: 17707. Waiting for deadlock...
Setup complete. Run 'gops stack <PID>' from another terminal, then send SIGINT (Ctrl+C) to stop.
G1: mu1 locked
G2: mu2 locked
G1: Attempting to lock mu2...
G2: Attempting to lock mu1...
```

由于死锁，导致程序 hang 住，并未退出，这正是 gops 一展身手进行并发诊断的最佳时机。诊断第一步就是查找当前目标主机上
运行的 Go 程序。

gops 可以列出所有可连接的 Go 进程及其 PID 和程序路径：

```shell
$gops
68    1     aTrustXtunnel  go1.20.5 /Library/sangfor/SDP/aTrust.app/Contents/Resources/bin/aTrustXtunnel
81    1     aTrustXtunnel  go1.20.5 /Library/sangfor/SDP/aTrust.app/Contents/Resources/bin/aTrustXtunnel
17707 10635 gops_deadlock* go1.24.3 /Users/tonybai/github/geekbang/column/go-advanced/part3/source/ch28/gops_deadlock/gops_deadlock
18680 15382 gops           go1.18.3 /Users/tonybai/Go/bin/gops
85163 85131 present        go1.24.3 /private/var/folders/cz/sbj5kg2d3m3c6j650z0qfm800000gn/T/go-build1258102410/b001/exe/present
85131 68928 go             go1.24.3 /Users/tonybai/.bin/go1.24.3/bin/go
```

当然，我们也可以用 gops tree 命令以树状形式展示可连接的 Go 进程：

```shell
$gops tree
...
├── 1
│   ├── 81 (aTrustXtunnel) {go1.20.5}
│   └── 68 (aTrustXtunnel) {go1.20.5}
├── 598
│   └── 62393 (net-core) {go1.23.0}
├── 10635
│   └── [*]  17707 (gops_deadlock) {go1.24.3}
├── 15382
│   └── 18706 (gops) {go1.18.3}
└── 68928
    └── 85131 (go) {go1.24.3}
        └── 85163 (present) {go1.24.3}
```

我们设计的目标程序在输出日志中打印了 pid(17707)，gops 也可以直接基于该 pid 进行后续诊断操作。gops 显示目标进程的
概要信息，包括父进程 pid、线程数量、CPU 使用、运行时间以及 gops agent 的连接方式等。这让我们可以对目标进程有一个
大致的了解：

```shell
$gops  17707     
parent PID:    10635
threads:    5
memory usage:    0.013%
cpu usage:    0.004%
username:    tonybai
cmd+args:    ./gops_deadlock
elapsed time:    01:14:44
local/remote:    127.0.0.1:53669 <-> :0 (LISTEN)
```

gops stats 显示指定进程的运行时统计信息，包括当前 goroutine 数量、GOMAXPROCS、内存分配、GC 暂停时间等。
这对于监控 goroutine 泄漏或 GC 压力非常有用。

```shell
$gops stats 17707
goroutines: 5
OS threads: 5
GOMAXPROCS: 8
num CPU: 8
```

gops version ：查看目标 Go 应用所使用的 Go 版本。

```shell
$gops version 17707 
go1.24.3
```

gops memstats ：打印目标进程的更详细的内存分配统计（runtime.MemStats）。

```shell
$gops memstats 17707  
alloc: 1.14MB (1193752 bytes)
total-alloc: 1.14MB (1193752 bytes)
sys: 7.96MB (8344840 bytes)
lookups: 0
mallocs: 386
frees: 13
heap-alloc: 1.14MB (1193752 bytes)
heap-sys: 3.75MB (3932160 bytes)
heap-idle: 2.02MB (2121728 bytes)
heap-in-use: 1.73MB (1810432 bytes)
heap-released: 1.99MB (2088960 bytes)
heap-objects: 373
stack-in-use: 256.00KB (262144 bytes)
stack-sys: 256.00KB (262144 bytes)
stack-mspan-inuse: 33.59KB (34400 bytes)
stack-mspan-sys: 47.81KB (48960 bytes)
stack-mcache-inuse: 9.44KB (9664 bytes)
stack-mcache-sys: 15.34KB (15704 bytes)
other-sys: 980.17KB (1003689 bytes)
gc-sys: 1.56MB (1638752 bytes)
next-gc: when heap-alloc >= 4.00MB (4194304 bytes)
last-gc: -
gc-pause-total: 0s
gc-pause: 0
gc-pause-end: 0
num-gc: 0
num-forced-gc: 0
gc-cpu-fraction: 0
enable-gc: true
debug-gc: false
```


回到重点问题上，如何用 gops 分析死锁，这就涉及 gops 的重要子命令 gops stack 了。通过 gops stack ，我们可以打印
指定 PID 的 Go 进程中所有 goroutine 的堆栈信息。这也是诊断死锁或 goroutine 卡在何处的极其有用的命令，其输出格式与
panic 时的堆栈类似，我们对 gops_deadlock 使用 gops stack 命令后的输出内容如下：


```shell
$gops stack 17707
goroutine 19 [running]:
runtime/pprof.writeGoroutineStacks({0x118ad880, 0xc000106060})
    /Users/tonybai/.bin/go1.24.3/src/runtime/pprof/pprof.go:764 +0x6a
runtime/pprof.writeGoroutine({0x118ad880?, 0xc000106060?}, 0x0?)
    /Users/tonybai/.bin/go1.24.3/src/runtime/pprof/pprof.go:753 +0x25
runtime/pprof.(*Profile).WriteTo(0xabfbcc0?, {0x118ad880?, 0xc000106060?}, 0x0?)
    /Users/tonybai/.bin/go1.24.3/src/runtime/pprof/pprof.go:377 +0x14b
github.com/google/gops/agent.handle({0x118ad858, 0xc000106060}, {0xc00008e000?, 0x1?, 0x1?})
    /Users/tonybai/Go/pkg/mod/github.com/google/gops@v0.3.28/agent/agent.go:200 +0x2992
github.com/google/gops/agent.listen({0xab239b8, 0xc000124040})
    /Users/tonybai/Go/pkg/mod/github.com/google/gops@v0.3.28/agent/agent.go:144 +0x1b4
created by github.com/google/gops/agent.Listen in goroutine 1
    /Users/tonybai/Go/pkg/mod/github.com/google/gops@v0.3.28/agent/agent.go:122 +0x35c

goroutine 1 [sleep]:
time.Sleep(0x12a05f200)
    /Users/tonybai/.bin/go1.24.3/src/runtime/time.go:338 +0x165
main.main()
    /Users/tonybai/github/geekbang/column/go-advanced/part3/source/ch28/gops_deadlock/main.go:60 +0x23d

goroutine 20 [sync.Mutex.Lock, 34 minutes]:
internal/sync.runtime_SemacquireMutex(0xc000106028?, 0xc0?, 0x1e?)
    /Users/tonybai/.bin/go1.24.3/src/runtime/sema.go:95 +0x25
internal/sync.(*Mutex).lockSlow(0xc0001040e8)
    /Users/tonybai/.bin/go1.24.3/src/internal/sync/mutex.go:149 +0x15d
internal/sync.(*Mutex).Lock(...)
    /Users/tonybai/.bin/go1.24.3/src/internal/sync/mutex.go:70
sync.(*Mutex).Lock(...)
    /Users/tonybai/.bin/go1.24.3/src/sync/mutex.go:46
main.main.func1()
    /Users/tonybai/github/geekbang/column/go-advanced/part3/source/ch28/gops_deadlock/main.go:29 +0x125
created by main.main in goroutine 1
    /Users/tonybai/github/geekbang/column/go-advanced/part3/source/ch28/gops_deadlock/main.go:23 +0x12b

goroutine 21 [sync.Mutex.Lock, 34 minutes]:
internal/sync.runtime_SemacquireMutex(0xc000106028?, 0x0?, 0x1e?)
    /Users/tonybai/.bin/go1.24.3/src/runtime/sema.go:95 +0x25
internal/sync.(*Mutex).lockSlow(0xc0001040e0)
    /Users/tonybai/.bin/go1.24.3/src/internal/sync/mutex.go:149 +0x15d
internal/sync.(*Mutex).Lock(...)
    /Users/tonybai/.bin/go1.24.3/src/internal/sync/mutex.go:70
sync.(*Mutex).Lock(...)
    /Users/tonybai/.bin/go1.24.3/src/sync/mutex.go:46
main.main.func2()
    /Users/tonybai/github/geekbang/column/go-advanced/part3/source/ch28/gops_deadlock/main.go:41 +0x125
created by main.main in goroutine 1
    /Users/tonybai/github/geekbang/column/go-advanced/part3/source/ch28/gops_deadlock/main.go:35 +0x193

goroutine 22 [sync.WaitGroup.Wait, 34 minutes]:
sync.runtime_SemacquireWaitGroup(0x0?)
    /Users/tonybai/.bin/go1.24.3/src/runtime/sema.go:110 +0x25
sync.(*WaitGroup).Wait(0x0?)
    /Users/tonybai/.bin/go1.24.3/src/sync/waitgroup.go:118 +0x48
main.main.func3()
    /Users/tonybai/github/geekbang/column/go-advanced/part3/source/ch28/gops_deadlock/main.go:50 +0x25
created by main.main in goroutine 1
    /Users/tonybai/github/geekbang/column/go-advanced/part3/source/ch28/gops_deadlock/main.go:49 +0x22c
```

如我们预期，gops stack 的输出清晰显示了有两个 goroutine 的堆栈顶部都处在sync.Mutex.Lock：

```shell
goroutine 20 [sync.Mutex.Lock, 34 minutes]:
goroutine 21 [sync.Mutex.Lock, 34 minutes]:
```

这类输出直接揭示了经典的 AB-BA 死锁模式：G1 持有 mu1 等待 mu2，G2 持有 mu2 等待 mu1。接下来，对照着源码查看，
便可以直接确认“死锁”逻辑的存在。

gops 还支持一些与性能调优问题有关的子命令，包括：
gops trace [duration]：获取指定进程在接下来一段时间（默认 5 秒） 的执行追踪数据（trace.out 文件），可以用 go tool trace 分析。
gops pprof-cpu [duration] / gops pprof-heap ：方便地获取 CPU profile 或 heap profile，并（可选地）启动
go tool pprof 进行分析。
gops gc ：手动触发一次 GC（主要用于调试，生产环境慎用）。

我们看到，gops 是一个非常轻量且适合快速查看线上 Go 进程的运行时状态快照的工具。当 gops 提供的快照信息不足以
完全理解并发行为，或者我们需要更持续的、可通过 Web 界面交互的剖析数据时，net/http/pprof 就派上用场了。