
---
sync.WaitGroup源码分析
---

本文重点讲解sync.WaitGroup 如何通过将计数器和等待者数量打包进一个 uint64 原子变量，配合信号量机制，实现高效的并发等待。


# 1 sync.WaitGroup的设计目标

sync.WaitGroup 是 Go 标准库提供的并发等待工具，用于等待一组 goroutine 全部完成。
主 goroutine 调用 Add 设置需要等待的数量，每个子 goroutine 完成后调用 Done，
主 goroutine 调用 Wait 阻塞直到所有子 goroutine 完成。

其核心保证包括：
计数准确：counter 精确反映未完成的 goroutine 数量，并发 Add/Done 不会丢失计数。
唤醒及时：counter 降到 0 的瞬间，所有阻塞在 Wait 上的 goroutine 都会被唤醒。
内存可见性：Done 的调用"synchronizes before"被它解除阻塞的 Wait 返回（符合 Go 内存模型）。


# 2 sync.WaitGroup的源码结构

```go
type WaitGroup struct {
    noCopy noCopy

    state atomic.Uint64 // 高 32 位 = counter（计数器），低 32 位 = waiter（等待者数量）
    sema  uint32        // 信号量，用于阻塞和唤醒 Wait 调用者
}
```

关键字段：
state：一个 64 位原子变量，将 counter 和 waiter 打包在一起，通过一次原子操作同时读写两个值。
sema：运行时信号量，Wait 通过 runtime_SemacquireWaitGroup 在此阻塞，Add 通过 runtime_Semrelease 在此唤醒。
noCopy：禁止复制标记，`go vet` 会检测到对 WaitGroup 的复制。

**为什么将 counter 和 waiter 打包在一个 uint64 中？**
如果分成两个独立的 uint32，在 Add 中需要先读 counter 再读 waiter（或反过来），两次读取之间可能被
其他 goroutine 修改，导致不一致。打包成一个 uint64 后，一次原子操作就能同时获取两个值的一致快照。


# 3 Add 的实现

```go
func (wg *WaitGroup) Add(delta int) {
    state := wg.state.Add(uint64(delta) << 32)
    v := int32(state >> 32)   // counter：高 32 位
    w := uint32(state)         // waiter：低 32 位
    if v < 0 {
        panic("sync: negative WaitGroup counter")
    }
    if w != 0 && delta > 0 && v == int32(delta) {
        panic("sync: WaitGroup misuse: Add called concurrently with Wait")
    }
    if v > 0 || w == 0 {
        return
    }
    // 此时 counter == 0 且 waiter > 0，需要唤醒所有等待者
    if wg.state.Load() != state {
        panic("sync: WaitGroup misuse: Add called concurrently with Wait")
    }
    wg.state.Store(0) // 重置 counter 和 waiter
    for ; w != 0; w-- {
        runtime_Semrelease(&wg.sema, false, 0) // 逐个唤醒等待者
    }
}
```

## 3.1 原子递增 counter

```go
state := wg.state.Add(uint64(delta) << 32)
```

delta 左移 32 位后加到 state 上，只影响高 32 位（counter），不影响低 32 位（waiter）。
Add 返回操作后的新 state 值，一次原子操作同时得到了 counter 和 waiter 的一致快照。

## 3.2 负值检查

```go
if v < 0 {
    panic("sync: negative WaitGroup counter")
}
```

counter 不允许为负数。如果 Done 调用次数多于 Add，会触发 panic。

## 3.3 误用检测

```go
if w != 0 && delta > 0 && v == int32(delta) {
    panic("sync: WaitGroup misuse: Add called concurrently with Wait")
}
```

这行检测一种特殊的误用场景：waiter > 0（有人在 Wait）的同时，counter 从 0 增加到 delta（说明
counter 曾经到达过 0 然后又被 Add 拉起来了），意味着 Add 和 Wait 并发使用了同一个已完成的 WaitGroup。

## 3.4 快速返回

```go
if v > 0 || w == 0 {
    return
}
```

两种情况可以直接返回：
- v > 0：counter 还没到 0，还有 goroutine 未完成，无需唤醒任何人。
- w == 0：没有等待者，即使 counter 为 0 也不需要唤醒。

## 3.5 唤醒所有等待者

```go
wg.state.Store(0)
for ; w != 0; w-- {
    runtime_Semrelease(&wg.sema, false, 0)
}
```

当 counter 恰好降到 0 且存在等待者时：
> 1 再次校验 state 未被并发修改（误用检测）。
> 2 将 state 归零（同时清除 counter 和 waiter），为下一轮复用做准备。
> 3 循环 w 次调用 runtime_Semrelease，逐个唤醒阻塞在信号量上的 Wait 调用者。


# 4 Done 的实现

```go
func (wg *WaitGroup) Done() {
    wg.Add(-1)
}
```

Done 就是 Add(-1) 的语法糖。没有任何额外逻辑。

这意味着 Done 的唤醒逻辑完全由 Add 处理——当 Add(-1) 将 counter 减到 0 时，
由调用 Done 的那个 goroutine 负责唤醒所有等待者。


# 5 Wait 的实现

```go
func (wg *WaitGroup) Wait() {
    for {
        state := wg.state.Load()
        v := int32(state >> 32)   // counter
        w := uint32(state)         // waiter
        if v == 0 {
            return // counter 已经是 0，无需等待
        }
        // CAS 递增 waiter 数量
        if wg.state.CompareAndSwap(state, state+1) {
            runtime_SemacquireWaitGroup(&wg.sema) // 阻塞在信号量上
            if wg.state.Load() != 0 {
                panic("sync: WaitGroup is reused before previous Wait has returned")
            }
            return
        }
    }
}
```

## 5.1 快速返回

```go
if v == 0 {
    return
}
```

如果 counter 已经是 0，说明所有 goroutine 都已完成，直接返回，不阻塞。

## 5.2 CAS 递增 waiter

```go
if wg.state.CompareAndSwap(state, state+1) {
```

用 CAS 将 waiter 加 1（state 的低 32 位加 1）。CAS 可能失败（被其他 goroutine 修改了 state），
失败后重新进入 for 循环，重新读取 state 重试。

**为什么 waiter 的递增需要 CAS 而不是直接 Add？**
因为在递增 waiter 之前需要检查 counter 是否为 0。如果先 Add(1) 再检查 counter，
可能出现：waiter 已经递增，但 counter 此时变成 0 并且唤醒逻辑已经执行完毕，
导致这个 waiter 永远不会被唤醒（信号丢失）。CAS 保证了"检查 counter + 递增 waiter"的原子性。

## 5.3 信号量阻塞

```go
runtime_SemacquireWaitGroup(&wg.sema)
```

CAS 成功后，调用运行时信号量阻塞当前 goroutine。当 Add 将 counter 减到 0 时，
会调用 runtime_Semrelease 唤醒阻塞在此的 goroutine。

## 5.4 唤醒后的复用检测

```go
if wg.state.Load() != 0 {
    panic("sync: WaitGroup is reused before previous Wait has returned")
}
```

被唤醒后，检查 state 是否已被归零。如果不为 0，说明在 Wait 返回之前，WaitGroup 又被 Add 了，
这属于误用（复用必须等所有 Wait 返回后才能开始新一轮 Add）。


# 6 并发场景下的执行流程

假设主 goroutine M 调用 wg.Add(3)，然后启动三个子 goroutine G1、G2、G3，M 调用 wg.Wait()：

初始状态：state = 0（counter=0，waiter=0）。

## 6.1 Add(3)
```
state.Add(3 << 32)
state = 0x0000000300000000  // counter=3, waiter=0
v=3, w=0 → v > 0，直接返回
```

## 6.2 M 调用 Wait()
```
state.Load() → counter=3, waiter=0
v=3, v != 0 → 不能快速返回
CAS(state, state+1) → state = 0x0000000300000001  // counter=3, waiter=1
runtime_SemacquireWaitGroup → M 阻塞
```

## 6.3 G1 调用 Done()（即 Add(-1)）
```
state.Add(-1 << 32)
state = 0x0000000200000001  // counter=2, waiter=1
v=2, v > 0 → 直接返回
```

## 6.4 G2 调用 Done()
```
state.Add(-1 << 32)
state = 0x0000000100000001  // counter=1, waiter=1
v=1, v > 0 → 直接返回
```

## 6.5 G3 调用 Done()（关键时刻）
```
state.Add(-1 << 32)
state = 0x0000000000000001  // counter=0, waiter=1
v=0, w=1 → 需要唤醒！
state.Store(0)              // 归零
runtime_Semrelease → 唤醒 M
```

## 6.6 M 被唤醒
```
state.Load() == 0 → 校验通过
Wait() 返回
```

**结果**：M 在所有子 goroutine 完成后被唤醒，流程正确。


# 7 state 的位布局设计

```
       高 32 位              低 32 位
┌──────────────────┬──────────────────┐
│   counter (v)    │   waiter (w)     │
│  goroutine 计数  │  等待者计数      │
└──────────────────┴──────────────────┘
         atomic.Uint64
```

## 7.1 为什么不用两个 atomic.Uint32？

如果 counter 和 waiter 是两个独立的原子变量，Add 的唤醒逻辑会出现竞争：

```go
// 假设分开存储的错误实现
func (wg *WaitGroup) Add(delta int) {
    new := atomic.AddInt32(&wg.counter, int32(delta))
    // ← 窗口期：其他 goroutine 可能在此时修改 waiter
    if new == 0 {
        w := atomic.LoadUint32(&wg.waiter) // 读到的 waiter 可能已过时
        for i := 0; i < int(w); i++ {
            runtime_Semrelease(&wg.sema)
        }
    }
}
```

在标注的窗口期内，Wait 可能刚好递增了 waiter 但还没来得及阻塞在信号量上，
此时 Add 已经读了旧的 waiter 值并完成了唤醒，导致后来的 Wait 永远阻塞。

打包成一个 uint64 后，Add 中一次原子操作同时得到 counter 和 waiter 的一致快照，消除了窗口期。

## 7.2 历史演进

Go 早期版本（1.18 之前）的 WaitGroup 使用 `[3]uint32` 数组来存储状态，需要手动处理 64 位对齐问题
（32 位平台上 uint64 不保证 64 位对齐，而原子操作需要对齐）。从 Go 1.20 开始改用 `atomic.Uint64`，
它内部自动保证对齐，代码大幅简化。


# 8 信号量机制

## 8.1 runtime_Semrelease 与 runtime_SemacquireWaitGroup

这两个函数是 Go 运行时提供的信号量原语，底层基于 treap（树堆）实现的等待队列：

```
runtime_SemacquireWaitGroup(&sema)
    → 检查 sema > 0？
      → 是：sema--，立即返回
      → 否：将当前 goroutine 加入 sema 的等待队列，挂起

runtime_Semrelease(&sema, false, 0)
    → sema++
    → 从等待队列中取出一个 goroutine，唤醒它
```

## 8.2 为什么用信号量而不是 channel？

channel 的实现本身依赖互斥锁，用它来实现 WaitGroup 会引入不必要的开销。
运行时信号量是更底层的原语，直接操作 goroutine 的调度状态，开销更小。
而且 WaitGroup 需要支持多个 Wait 调用者同时等待，信号量天然支持这种一对多唤醒模式。


# 9 常见误用与 panic 场景

## 9.1 counter 变为负数

```go
var wg sync.WaitGroup
wg.Add(1)
wg.Done()
wg.Done() // panic: sync: negative WaitGroup counter
```

Done 次数超过 Add，counter 变为负数。

## 9.2 Add 与 Wait 并发误用

```go
var wg sync.WaitGroup
go func() {
    wg.Add(1) // 在 Wait 之后才 Add → 可能 panic
    // ...
    wg.Done()
}()
wg.Wait()
```

正确做法是在启动 goroutine 之前调用 Add：
```go
var wg sync.WaitGroup
wg.Add(1)
go func() {
    defer wg.Done()
    // ...
}()
wg.Wait()
```

## 9.3 未等 Wait 返回就复用

```go
var wg sync.WaitGroup
wg.Add(1)
go func() { wg.Done() }()
wg.Wait()
// 如果紧接着在另一个 goroutine 的 Wait 尚未返回时就 Add
wg.Add(1) // 可能 panic: WaitGroup is reused before previous Wait has returned
```

复用 WaitGroup 必须确保前一轮的所有 Wait 都已返回。


# 10 总结

sync.WaitGroup 的实现通过巧妙的位打包和信号量配合，用极少的代码实现了高效的并发等待：

状态打包：counter 和 waiter 打包在一个 atomic.Uint64 中，一次原子操作获取一致快照，消除读取窗口期。
Add/Done：原子操作递增/递减 counter，counter 归零时负责唤醒所有等待者。
Wait：CAS 递增 waiter 后阻塞在信号量上，保证"检查 counter + 注册等待"的原子性。
信号量：使用运行时信号量而非 channel，开销更小，天然支持一对多唤醒。

整体设计高度紧凑——仅一个 uint64 和一个 uint32 就承载了全部同步逻辑，没有互斥锁，
热路径上仅需一次原子操作。这体现了 Go 标准库对最小化同步开销的极致追求。
