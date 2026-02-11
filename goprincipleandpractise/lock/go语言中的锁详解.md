
---
go语言中的锁详解
---

# 1 锁的使用场景
什么时候需要使用锁？
答案是产生数据竞争(data race)时，在并发读写中为了保证数据正确性，需要使用锁，例如多个协程并发读写同一个
string、map、slice、struct等。
使用锁虽然能保证数据的正确性，但是会引起程序性能的大幅退化，因为大量的锁等待和加解锁的开销会耗费大量时间。
那么，如何检测data race呢？

```shell
go build -race xxx.go
go run -race xxx.go
go test -race xxx.go
```
注意:我们可以在单元测试，压力测试或者日常开发调试时使用上述指令检测程序是否存在data race，但在生产环境
千万不能这样做，因为data race检测会带来10倍以上的性能开销，对线上环境影响太大。

# 2 使用锁的最佳实践
要尽量避免使用锁带来的性能退化，我们有以下几个思路：

## 2.1 缩小临界区
在使用锁时，我们为了避免忘记释放锁，一般会使用defer来释放锁，但这样会导致锁的临界区扩大到函数结束；
但如果我们在执行完需要锁保护的操作后(通常是写操作)及时释放锁，便可缩小锁的临界区，提升程序的性能。
当然，前提是我们能保证程序的正确性，譬如在代码比较长的情况下，直接释放锁的话，如果新增了逻辑分支代码
可能会遗漏解锁，此时使用defer会更可靠。


benchmark 测试代码详见 [performance/narrow-critical-space](performance/narrow-critical-space)

核心思路：`countDefer` 使用 `defer m.Unlock()` 导致 `time.Sleep` 也被包含在临界区内；
`countNarrow` 在 `c.i++` 后立即 `m.Unlock()`，`time.Sleep` 在锁外执行。

```shell
go test -bench=^Bench -benchtime=5s -count=5 -benchmem .
```

| 用例 | 均值(ns/op) | 对比 |
|---|---:|---|
| `BenchmarkCountDefer` | 15,934 | 基线 |
| `BenchmarkCountNarrow` | 1,645 | **快约 9.69x（延迟降低约 89.7%）** |

![narrow-critical-space](images/narrow-critical-space.png)

## 2.2 减小锁的粒度
具体来讲，就是使用分段锁，将一把全局大锁替换为多个分段锁，减小锁的粒度，这样便能大幅减少锁竞争，通过
空间换时间的方式提升程序性能。

benchmark 测试代码详见 [performance/segment-lock-replace-global-lock](performance/segment-lock-replace-global-lock)

```shell
go test -bench=^Bench -benchtime=5s -count=5 -benchmem .
```

| 读写比 | 全局锁LM(ns/op) | 分段锁SM(ns/op) | 对比 |
|---|---:|---:|---|
| 9:1（读多） | 97.680 | 43.156 | **SM快约 2.26x（-55.8%）** |
| 1:9（写多） | 98.118 | 55.146 | **SM快约 1.78x（-43.8%）** |
| 5:5（均衡） | 64.938 | 50.182 | **SM快约 1.29x（-22.7%）** |

![segment-lock](images/segment-lock.png)

从这组数据看，三种负载下分段锁都更快，且**读多场景提升反而最大**。这并不与"分段锁主要缓解写争用"矛盾，
原因有三层：

1. **全局 RWMutex 的 `readerCount` 是隐藏热点**：RLock/RUnlock 每次都对同一个 `readerCount` 做 `atomic.Add`，
   10 核并发时 MESI 协议导致 cache line 在核间反复迁移（cache line bouncing）。这个硬件层面的争用与"读不阻塞读"
   的逻辑语义无关——分段后将这一热点分散到 32 个独立的 `readerCount`，效果立竿见影。
2. **分段对读的并行度提升 > 对写的并行度提升**：10 个 goroutine + 32 个 shard，偶发碰撞（两个 goroutine 命中同一 shard）时，
   读 vs 读可并行通过（RLock 不互斥），写 vs 写仍然串行（exclusive Lock 互斥）。分段后读路径在 shard 间并行、shard 内也并行；
   写路径在 shard 间并行、**shard 内仍然串行**。
3. **写路径的固定开销更高**：map 写入（可能触发 hash 扩容）比 map 读取贵，exclusive Lock/Unlock 的状态机比 RLock/RUnlock 复杂。
   这些"固定开销"无法被分段优化掉，稀释了分段带来的相对收益。

结论：**分段锁的核心收益是将锁热点从一把全局锁分散到 N 个分片锁，读写两条路径都受益；
在纯内存操作（无 Sleep）的基准模型下，读路径因 `readerCount` cache line 争用的消除 + shard 内读读并行而获益更大。**


## 2.3 读写分离
在读多写少的场景，采用读写分离对性能提升最为明显，其核心思路是读写和写写是互斥的，但读读可以并发执行，相比
互斥锁所有操作都互斥，读写锁可以减少锁竞争，提升程序性能。

benchmark 测试代码详见 [performance/rw-lock-replace-mutex](performance/rw-lock-replace-mutex)

```shell
go test -bench=^Bench -benchtime=5s -count=5 -benchmem .
```

| 读写比 | Mutex(ns/op) | RWMutex(ns/op) | 对比 |
|---|---:|---:|---|
| 9:1（读多） | 3,712.0 | 1,047.8 | **RWMutex快约 3.54x（-71.8%）** |
| 1:9（写多） | 3,690.2 | 3,687.4 | 基本持平 |
| 5:5（均衡） | 3,673.0 | 3,281.8 | **RWMutex快约 1.12x（-10.6%）** |

![rw-replace-mutex](images/rw-replace-mutex.png)

**RWMutex 为什么不会出现 writer 饥饿？**

Go 的 `sync.RWMutex` 内部有防饥饿机制：当有 writer 在等待时，新的 reader 会被阻塞（不能继续获取读锁），
确保 writer 不会被源源不断的 reader 饿死。实现上是通过 `readerCount` 减去一个极大值
（`rwmutexMaxReaders = 1 << 30`）来通知后续 reader "有 writer 在排队"。


## 2.4 使用atomic代替锁实现无锁化
如果只是在并发操作时保护一个变量，使用原子操作比使用互斥锁性能更优。
atomic 主要由 CPU 原子指令完成；Mutex 在无竞争时走用户态快路径，在竞争时会走慢路径并涉及阻塞/唤醒，开销通常更高。

benchmark 测试代码详见 [performance/atomic-replace-mutex](performance/atomic-replace-mutex)


```shell
go test -bench=^Bench -benchtime=5s -count=5 -benchmem .
```

| 用例 | 均值(ns/op) | 说明 |
|---|---:|---|
| `BenchmarkAddNormal` | 0.057 | 无同步基线（非并发安全） |
| `BenchmarkAddUseAtomic` | 31.930 | 原子累加 |
| `BenchmarkAddUseMutex` | 62.298 | 互斥锁累加 |

对比可见：在“单变量并发计数”场景下，atomic 相比 mutex 约快 **1.95x**（延迟降低约 **48.7%**）。

![atomic-replace-mutex](images/atomic-replace-mutex.png)

### 2.4.1 atomic 操作全景

| 操作 | 语义 | 典型场景 |
|------|------|---------|
| `atomic.Load` / `Store` | 原子读 / 原子写 | 读写一个 flag 或配置值 |
| `atomic.Add` | 原子加减 | 计数器 |
| `atomic.CompareAndSwap` (CAS) | 如果当前值==old 则写入new | 无锁数据结构、乐观更新 |
| `atomic.Swap` | 原子交换，返回旧值 | 状态切换 |
| `atomic.Value` | 原子存取任意类型 | 热更新配置（存只读对象） |

**Go 1.19+ 类型化原子操作（推荐）：**

```go
// 旧写法
var counter int64
atomic.AddInt64(&counter, 1)
val := atomic.LoadInt64(&counter)

// 新写法（Go 1.19+）—— 类型安全，不需要传指针
var counter atomic.Int64
counter.Add(1)
val := counter.Load()

// 还有 atomic.Bool、atomic.Pointer[T] 等
var flag atomic.Bool
flag.Store(true)

var cfg atomic.Pointer[Config]
cfg.Store(&Config{Port: 8080})
c := cfg.Load()  // *Config
```

### 2.4.2 CAS 自旋模式

CAS 是构建无锁并发的核心原语。它的典型模式是"读-计算-CAS 写回"循环：

```go
// 无锁地将 counter 乘以 2
for {
    old := atomic.LoadInt64(&counter)
    new := old * 2
    if atomic.CompareAndSwapInt64(&counter, old, new) {
        break  // CAS 成功，退出
    }
    // CAS 失败说明有其他 goroutine 修改了 counter，重试
}
```

CAS 适合竞争不激烈的场景。如果竞争激烈，CAS 循环会大量重试，性能反而不如 Mutex。

# 3 使用锁的避坑指南

## 3.1 锁是不能拷贝的

```shell
grep -h 'must not be copied' $(go env GOROOT)/src/sync/*.go
// A Cond must not be copied after first use.
// noCopy may be embedded into structs which must not be copied
// The zero Map is empty and ready for use. A Map must not be copied after first use.
// A Mutex must not be copied after first use.
// A Once must not be copied after first use.
// A Pool must not be copied after first use.
// A RWMutex must not be copied after first use.
// A WaitGroup must not be copied after first use.
```
可以看到，标准库sync里的数据结构都是不能拷贝的，如果拷贝锁，就是拷贝了状态，等同于使用了新锁，那就是在并发场景
下使用不同的锁来保护全局变量，其结果是无法保证数据的正确性。

譬如下面这个案例：

```golang
package main

import (
	"fmt"
	"sync"
	"time"
)

var num int

func addWrong(m sync.Mutex) {
	m.Lock()
	defer m.Unlock()
	for i := 0; i < 1000; i++ {
		num++
		time.Sleep(time.Microsecond)
	}
}

func addRight(m *sync.Mutex) {
	m.Lock()
	defer m.Unlock()
	for i := 0; i < 1000; i++ {
		num++
		time.Sleep(time.Microsecond)
	}
}

func main() {
	var m sync.Mutex
	go addWrong(m)
	go addWrong(m)
	//go addRight(&m)
	//go addRight(&m)
	time.Sleep(time.Second * 2)
	fmt.Println("num = ", num)
}
```

如果拷贝锁，使用go vet检测代码会报拷贝锁的提醒
```shell
go vet        
# go-notes/lock/trap/no-copy-of-mutex
./main.go:11:17: addWrong passes lock by value: sync.Mutex
./main.go:31:14: call of addWrong copies lock value: sync.Mutex
./main.go:32:14: call of addWrong copies lock value: sync.Mutex

```
如果执行代码，会发现执行结果与预期不一致，无法保证数据正确性，每次执行结果可能都不一样。





![copy-mutex.png](images%2Fcopy-mutex.png)





解决的方法很简单，不要拷贝锁，传递锁的引用(指针)就好了。





![right-use-of-mutex.png](images%2Fright-use-of-mutex.png)





## 3.2 标准库sync里的锁是不可重入的，所以不要重复加锁，以免造成死锁。

```golang
package main

import (
	"fmt"
	"sync"
)

func HelloWorld(m *sync.Mutex) {
	m.Lock()
	defer m.Unlock()
	fmt.Println("Hello")
	m.Lock()
	defer m.Unlock()
	fmt.Println("World")
}

func helloWorld(m *sync.Mutex) {
	m.Lock()
	fmt.Println("Hello")
	m.Unlock()
	m.Lock()
	fmt.Println("World")
	m.Unlock()
}

func main() {
	var m sync.Mutex
	HelloWorld(&m)
	//helloWorld(&m)
}

```

执行代码会出现死锁bug。

```shell
 go run main.go
Hello
fatal error: all goroutines are asleep - deadlock!

goroutine 1 [semacquire]:
sync.runtime_SemacquireMutex(0xc000124008, 0x10, 0x1)
        /usr/local/go/src/runtime/sema.go:71 +0x25
sync.(*Mutex).lockSlow(0xc00012a008)
        /usr/local/go/src/sync/mutex.go:138 +0x165
sync.(*Mutex).Lock(...)
        /usr/local/go/src/sync/mutex.go:81
main.HelloWorld(0xc00012a008)
        /Users/qiujun/go/src/go-notes/lock/trap/no-reentry-mutex/main.go:22 +0xed
main.main()
        /Users/qiujun/go/src/go-notes/lock/trap/no-reentry-mutex/main.go:38 +0x2a
exit status 2

```

这里出现死锁的原因在于标准库sync的互斥锁Mutex(包括读写锁)是不可重入的，重复加锁之前这个锁必须是已经释放
了才可以，本案例中释放锁的操作根据defer语法是后进先出(执行)，所以第二次加锁时，第一次加的锁还未释放，
因为它还在等待第二次的defer操作释放锁，而第二次加锁由于第一次的锁还未释放掉所以无法加锁成功，会一直阻塞，
等待第一次锁的释放，最终导致循环等待，出现死锁的bug。

解决的方案是不使用defer，这样便可顺序加锁和释放锁，但是这个问题的关键在于互斥锁Mutex是不可重入的，所以最好
不要重复加锁。

![solve-repeat-mutex](images/solve-repeat-mutex.png)

## 3.3 atomic.Value误用导致程序崩溃
通常我们会使用atomic.Value来确保更新配置的并发安全，但如果我们配置里使用的是无法保证线程安全的map，那么有可能
出现多个协程并发的去读写配置，出现并发读写map的问题导致程序崩溃。
所以，使用atomic.Value需要注意:
虽然atomic.Value可以实现对任何类型(包括自定义类型)数据的原子操作(读写操作)，但是最好不要使用atomic.Value
存储引用类型的值，这样可能会导致数据不是并发安全的。
因为atomic.Value内部实际上维护的是存储值的指针，而这个指针因为不对外暴露，所以认为是并发安全的。然而如果
尝试用它来存储引用类型，维护的就是这个引用类型的指针，则不能保证实际的数据是并发安全的。
对于一个引用类型，我们实际上只是Store了一个指针，只是对一个指针的原子操作，而这个指针实际指向的地址的值，并不在
atomic.Value的保护下，所以并不是并发安全的。

简言之，atomic.Value只保证存取对象时是并发安全的，并不保证存取的对象本身的操作是并发安全的。所以，要么存放
只读对象，要么对象自身的操作集合必须是并发安全的。

另外:
Store写入的数据不能是空指针nil；
对于同一个atomic.Value不能存入不同类型的值。

## 3.4 更多死锁模式

除了上面的重入死锁，还有几种常见地死锁模式需要警惕：

### 3.4.1 锁顺序死锁

两个 goroutine 以相反顺序获取两把锁，形成循环等待：

```go
var mu1, mu2 sync.Mutex

// goroutine A              // goroutine B
mu1.Lock()                  mu2.Lock()
mu2.Lock()  // 等 B 释放    mu1.Lock()  // 等 A 释放 → 死锁！
```

**解决方案**：全局统一锁的获取顺序。如果业务上必须同时持有 mu1 和 mu2，所有代码路径都先锁 mu1 再锁 mu2。

### 3.4.2 RWMutex 读锁内获取写锁

```go
var rw sync.RWMutex

rw.RLock()
// ... 发现需要写入 ...
rw.Lock()  // 死锁！当前 goroutine 持有读锁，写锁要等所有读锁释放
```

Go 的 RWMutex 不支持锁升级（read lock → write lock）。如果需要"先读后写"，
必须先释放读锁再获取写锁，或者直接用写锁。

### 3.4.3 持有锁时阻塞在 channel

```go
mu.Lock()
ch <- data   // 如果 channel 满了，阻塞在这里，锁一直不释放
mu.Unlock()  // 永远执行不到
```

**原则**：不要在持有锁的情况下做可能阻塞的操作（channel 收发、网络 IO、等待其他锁）。

## 3.5 对未加锁的 Mutex 调用 Unlock 会 panic

```go
var mu sync.Mutex
mu.Unlock()  // panic: sync: unlock of unlocked mutex
```

这是 Mutex 的保护机制，防止不对称的 Lock/Unlock。在生产代码中，确保每个 `Lock()` 都有对应的 `Unlock()`，
最简单的方式就是 `defer mu.Unlock()`。

# 4 sync.Map：标准库的并发安全 Map

第 2.2 节用分段锁实现了并发安全的 Map，Go 标准库也提供了 `sync.Map`，但二者的适用场景不同。

## 4.1 sync.Map 的内部机制

`sync.Map` 内部维护两个 map：`read`（只读，无锁访问）和 `dirty`（需要加锁）。

```
读操作：先查 read map（无锁 atomic），命中直接返回；未命中再加锁查 dirty
写操作：加锁写入 dirty map
晋升：当 read miss 次数超过 dirty 长度时，dirty 晋升为新的 read（一次性操作）
```

## 4.2 sync.Map vs 分段锁（concurrent-map）vs RWMutex+map

| | sync.Map | 分段锁 | RWMutex + map |
|---|---|---|---|
| 适用场景 | key 集合稳定后主要读；或各 goroutine 操作不同 key 子集 | 通用读写，尤其是写入频繁 | 简单场景，数据量不大 |
| 读性能 | 极高（read map 无锁） | 高（只锁对应分片） | 高（读锁并发） |
| 写性能 | 较差（频繁写导致 dirty→read 晋升开销） | 高（只锁对应分片） | 中（写锁独占） |
| 内存开销 | 较高（两份 map） | 中（N 个分片 map） | 低（一份 map） |
| 类型安全 | 否（key/value 都是 `any`） | 可泛型化 | 可泛型化 |

**实践建议**：
- 读远多于写（如配置缓存）→ `sync.Map`
- 读写均衡或写多 → 分段锁
- 数据量小、并发不高 → `RWMutex + map` 最简单

# 5 TryLock（Go 1.18+）

Go 1.18 为 `sync.Mutex` 和 `sync.RWMutex` 新增了非阻塞获取锁的方法：

```go
var mu sync.Mutex

if mu.TryLock() {
    defer mu.Unlock()
    // 获取到锁，执行操作
} else {
    // 锁被其他 goroutine 持有，走降级/跳过逻辑
}

// RWMutex 同理
var rw sync.RWMutex
rw.TryRLock()   // 非阻塞尝试获取读锁
rw.TryLock()    // 非阻塞尝试获取写锁
```

**适用场景**：
- 限流/降级：如果获取不到锁，说明有其他操作正在进行，直接返回而不阻塞
- 避免死锁：在复杂的多锁场景中，TryLock 失败就释放已持有的锁，重试
- 探测性操作：检查某个资源是否正在被使用

**注意**：TryLock 不参与饥饿模式的公平队列，不保证最终能获取到锁。不要用 `for { if mu.TryLock() ... }` 自旋，这比直接 `mu.Lock()` 性能更差。

# 6 锁竞争的诊断：mutex profile

`-race` 检测的是数据竞争（data race），而 **mutex profile** 检测的是锁等待热点（哪些锁竞争最激烈、等待时间最长）。

```go
import "runtime"

func init() {
    // 设置采样率：每 N 次锁竞争事件采样 1 次
    // 设为 1 表示全量采样（开销较大，适合测试环境）
    // 设为 5-10 适合线上低开销采样
    runtime.SetMutexProfileFraction(5)
}
```

然后通过 pprof 查看：

```shell
# 如果程序暴露了 pprof HTTP 端点
go tool pprof http://localhost:6060/debug/pprof/mutex

# 或在 benchmark/测试中
go test -mutexprofile=mutex.prof -bench=.
go tool pprof mutex.prof
```

在 pprof 中，`contentions` 表示锁竞争次数，`delay` 表示总等待时间。
排名靠前的调用栈就是锁竞争的热点，优先考虑对这些位置做缩小临界区、分段锁等优化。

# 7 Mutex 内部实现原理

> 详见 [../sync/Mutex互斥锁实现原理.md](../sync/Mutex互斥锁实现原理.md)

这里概括关键点：

**状态字段 state 的位域设计**

```
 ┌───────────────────────────────┬──────────┬───────┬────────┐
 │         waiter (29 bit)       │ starving │ woken │ locked │
 └───────────────────────────────┴──────────┴───────┴────────┘
   等待获取锁的 goroutine 数量     饥饿模式   已唤醒   已锁定
```

**正常模式 vs 饥饿模式**

| | 正常模式 | 饥饿模式 |
|---|---|---|
| 触发条件 | 默认 | 某 goroutine 等待超过 1ms |
| 锁获取方式 | 新来的 goroutine 与被唤醒的 goroutine 竞争（新来的有优势） | 锁直接交给队首等待者（FIFO） |
| 自旋 | 允许（多核 + 自旋次数 ≤ 4 + 非饥饿） | 禁止 |
| 性能 | 高吞吐 | 低吞吐但公平 |
| 退出条件 | — | 等待者等待时间 < 1ms 或是最后一个等待者 |

这种双模式设计保证了：低竞争时高性能，高竞争时不饿死。
