
---
sync.Once源码分析
---

本文重点讲解sync.Once 如何通过快慢路径和双重检查，兼顾了性能和并发安全性。


# 1 sync.Once的设计目标

sync.Once 是 Go 标准库提供的一个工具，用于确保某个函数（f）在并发环境下只被执行一次，并且在函数返回时，
所有调用者都能感知到执行已完成。它的典型使用场景包括单例模式初始化、一次性资源加载等。

其核心保证包括：
只执行一次：即使被多个 goroutine 同时调用，f 只会被执行一次。
同步完成：所有调用 Do 的 goroutine 在返回时，f 已执行完毕。



# 2 sync.Once的源码结构

```go
type Once struct {
    done uint32      // 标记函数是否已执行，0 表示未执行，1 表示已执行
    m    sync.Mutex // 互斥锁，用于慢路径的同步
}

func (o *Once) Do(f func()) {
    if o.done.Load() == 0 { // 第一次检查（快路径）
        o.doSlow(f)         // 如果未执行，进入慢路径
    }
}

func (o *Once) doSlow(f func()) {
    o.m.Lock()          // 加锁
    defer o.m.Unlock()  // 确保解锁
    if o.done.Load() == 0 { // 第二次检查（慢路径）
        defer o.done.Store(1) // 确保 f 执行完后标记为已完成
        f()                   // 执行用户函数
    }
}
```

关键字段
done：使用 uint32 类型，通过原子操作（如 Load 和 Store）记录执行状态。
m：sync.Mutex 类型的互斥锁，用于保护慢路径中的竞争。


# 3 实现原理

sync.Once 的实现分为快路径和慢路径，通过两次检查 o.done.Load() == 0，结合原子操作和互斥锁，实现了高效
且安全的“只执行一次”逻辑。

## 3.1 快路径: 第一次检查

```go
if o.done.Load() == 0 {
    o.doSlow(f)
}
```

作用：快速判断函数是否已执行。

实现：
使用 atomic.LoadUint32（封装在 o.done.Load() 中）原子读取 done 的值。
如果 done == 0，表示 f 尚未执行，进入慢路径 doSlow。
如果 done == 1，表示 f 已执行，直接返回，不调用 doSlow。

性能优化：
快路径无需加锁，仅依赖原子操作，减少了锁竞争的开销。
大多数后续调用（f 已执行后）会命中 done == 1，直接返回，效率极高。

## 3.2 慢路径: 第二次检查

```go
func (o *Once) doSlow(f func()) {
    o.m.Lock()
    defer o.m.Unlock()
    if o.done.Load() == 0 {
        defer o.done.Store(1)
        f()
    }
}
```

作用：在并发情况下，确保只有一个 goroutine 执行 f，其他 goroutine 等待其完成。

**实现**
> 1 加锁：通过 o.m.Lock() 获取互斥锁，保证只有一个 goroutine 进入临界区。
> 2 第二次检查：再次使用 o.done.Load() 检查 done 是否为 0。
      如果 done == 0，执行 f()，并在返回前通过 o.done.Store(1) 标记为已完成。
      如果 done == 1，直接退出（无需执行 f）。

> 3 解锁：defer o.m.Unlock() 确保锁被释放。

延迟标记：defer o.done.Store(1) 确保 f() 完全执行后才更新 done，满足“所有调用者在返回时 f 已完成”的保证。

# 4 为什么需要判断两次o.done.Load() == 0 ？

## 4.1 假设直接加锁判断一次

假设 sync.Once 的实现如下：

```go
func (o *Once) Do(f func()) {
    o.m.Lock()
    defer o.m.Unlock()
    if o.done.Load() == 0 {
        f()
        o.done.Store(1)
    }
}
```


这样做会存在两个问题:

> 性能开销：

每次调用 Do 都会加锁，即使 f 已执行完毕（done == 1）。
互斥锁的加锁和解锁操作涉及内核态切换，开销远高于原子操作。
在高并发场景下，所有 goroutine 都会竞争锁，导致严重的性能瓶颈。

> 不必要的同步：

当 f 已执行后，后续调用只需快速返回，但加锁会强制同步，浪费资源。


**结论**
直接加锁的方式虽然逻辑简单，但无法区分“首次执行”和“已完成”的场景，导致性能低下。

## 4.2 两次检查的优势

现有实现通过双重检查（Double-Checked Locking），结合快慢路径，解决了性能和安全问题：

快路径优化：
第一次 o.done.Load() == 0 使用原子操作，无锁开销。
如果 done == 1（已执行），直接返回，避免进入慢路径。
这适用于大多数后续调用（f 已完成的情况），极大提升了性能。

慢路径安全性：
当多个 goroutine 同时首次调用 Do 时，第一次检查会让它们都进入 doSlow。
第二次检查在锁保护下，确保只有一个 goroutine 执行 f，其他 goroutine 等待锁释放。
锁释放后，其他 goroutine 看到 done == 1，直接返回，保证了“只执行一次”和“同步完成”。

避免竞争条件：
如果只有第一次检查而无锁，可能出现多个 goroutine 同时通过检查并执行 f 的情况。
第二次检查加锁消除了这种竞争，确保并发安全。


# 5 并发场景下的执行流程

假设三个 goroutine（G1、G2、G3）同时调用 o.Do(f)：

初始状态：o.done == 0。

## 5.1 快路径：
G1、G2、G3 同时执行 o.done.Load() == 0，都通过检查，进入 doSlow。

## 5.2 慢路径竞争：
G1 抢到锁，进入 doSlow，第二次检查 o.done == 0，执行 f()。
G2、G3 等待锁。

## 5.3 G1 执行完成：
f() 返回，o.done.Store(1) 将 done 设为 1，G1 释放锁。

## 5.4 G2、G3 继续：
G2 抢到锁，第二次检查 o.done == 1，不执行 f，释放锁。
G3 同理。

**结果**
f 只执行一次，所有 goroutine 返回时 f 已完成。


# 6 done 为什么是第一个字段
字段 done 的注释也非常值得一看：

```go
type Once struct {
    // done indicates whether the action has been performed.
    // It is first in the struct because it is used in the hot path.
    // The hot path is inlined at every call site.
    // Placing done first allows more compact instructions on some architectures (amd64/x86),
    // and fewer instructions (to calculate offset) on other architectures.
    done uint32
    m    Mutex
}
```

其中解释了为什么将 done 置为 Once 的第一个字段：done 在热路径中，done 放在第一个字段，能够减少 CPU 指令，也就是说，这样做能够提升性能。

简单解释下这句话：

热路径(hot path)是程序非常频繁执行的一系列指令，sync.Once 绝大部分场景都会访问 o.done，在热路径上是比较好理解的，如果 hot path 编译后
的机器码指令更少，更直接，必然是能够提升性能的。

为什么放在第一个字段就能够减少指令呢？因为结构体第一个字段的地址和结构体的指针是相同的，如果是第一个字段，直接对结构体的指针解引用即可。
如果是其他的字段，除了结构体指针外，还需要计算与第一个值的偏移(calculate offset)。在机器码中，偏移量是随指令传递的附加值，CPU 需要
做一次偏移值与指针的加法运算，才能获取要访问的值的地址。因为，访问第一个字段的机器代码更紧凑，速度更快。

# 7 总结

sync.Once 的实现原理通过快慢路径和双重检查，兼顾了性能和并发安全性：
快路径（第一次检查）：使用原子操作快速判断是否已执行，优化后续调用。
慢路径（第二次检查）：用锁保护首次执行，确保“只执行一次”和“同步完成”。

两次检查的原因：第一次避免不必要的锁开销，第二次在锁内消除竞争条件。
直接加锁判断一次虽然可行，但会丧失性能优势，尤其在高并发场景下效率低下。
现有设计是性能与安全的完美平衡，体现了 Go 运行时对并发优化的深思熟虑。