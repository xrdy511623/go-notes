
---
sync.Pool源码分析
---

本文重点讲解sync.Pool 如何通过 per-P 本地缓存、无锁队列和 victim cache 机制，在高并发场景下实现高效的对象复用。


# 1 sync.Pool的设计目标

sync.Pool 是 Go 标准库提供的临时对象池，用于缓存已分配但暂时不用的对象，供后续复用，从而减少堆分配次数，
降低 GC 压力。它的典型使用场景包括 buffer 复用、编解码器复用等高频分配场景。

其核心特性包括：
并发安全：多个 goroutine 可以同时调用 Get 和 Put，无需额外加锁。
临时性：Pool 中的对象可能在任何时候被 GC 清除，不保证持久性。
高性能：热路径上无锁操作，避免了锁竞争带来的开销。


# 2 sync.Pool的源码结构

## 2.1 核心数据结构

```go
type Pool struct {
    noCopy noCopy

    local     unsafe.Pointer // per-P 本地池，实际类型是 [P]poolLocal
    localSize uintptr        // local 数组的长度

    victim     unsafe.Pointer // 上一轮 GC 的 local（受害者缓存）
    victimSize uintptr        // victim 数组的长度

    New func() any // 当 Get 返回 nil 时，调用 New 创建新对象
}
```

关键字段：
local：per-P 的本地缓存数组，每个 P 有自己独立的 poolLocal，热路径上无锁。
victim：上一轮 GC 周期的 local 副本，用于平滑 GC 清理的冲击（victim cache 机制）。
New：用户提供的工厂函数，当池中无可用对象时调用。

## 2.2 per-P 本地存储

```go
type poolLocal struct {
    poolLocalInternal

    // 填充到 128 字节，防止 false sharing
    pad [128 - unsafe.Sizeof(poolLocalInternal{})%128]byte
}

type poolLocalInternal struct {
    private any       // 只能被当前 P 使用，无需任何同步
    shared  poolChain // 当前 P 可以 pushHead/popHead；其他 P 只能 popTail
}
```

每个 P 拥有一个 poolLocal，包含两个存储位置：
private：私有槽位，只有当前 P 能访问，存取都是直接赋值，零同步开销。
shared：共享双端队列，当前 P 从头部操作（无锁），其他 P 从尾部偷取（CAS 操作）。
pad：填充字段，确保不同 P 的 poolLocal 不会落在同一个 CPU 缓存行上，避免 false sharing。


# 3 Put 的实现

```go
func (p *Pool) Put(x any) {
    if x == nil {
        return
    }
    l, _ := p.pin()
    if l.private == nil {
        l.private = x        // 优先放入 private 槽位
    } else {
        l.shared.pushHead(x) // private 已满，放入 shared 队列头部
    }
    runtime_procUnpin()
}
```

执行流程：

> 1 空值检查：如果 x == nil，直接返回。

> 2 pin 当前 P：调用 p.pin() 将当前 goroutine 绑定到 P（禁止抢占），获取当前 P 对应的 poolLocal。

> 3 优先 private：如果 private 为空，直接存入 private，这是最快的路径（直接赋值，无任何同步）。

> 4 回退 shared：如果 private 已有对象，放入 shared 队列的头部。pushHead 是单生产者操作，当前 P 独占，同样无锁。

> 5 unpin：恢复抢占。


# 4 Get 的实现

```go
func (p *Pool) Get() any {
    l, pid := p.pin()
    x := l.private   // 第一步：从 private 取
    l.private = nil
    if x == nil {
        x, _ = l.shared.popHead()   // 第二步：从本地 shared 头部取
        if x == nil {
            x = p.getSlow(pid)       // 第三步：进入慢路径
        }
    }
    runtime_procUnpin()
    if x == nil && p.New != nil {
        x = p.New()                  // 第四步：调用 New 创建新对象
    }
    return x
}
```

Get 的查找顺序体现了"从近到远"的优化策略：

## 4.1 第一步：private 槽位

直接取 private，置空。这是最快的路径，无任何同步操作。

## 4.2 第二步：本地 shared 队列头部

从当前 P 的 shared 队列头部弹出。popHead 是单消费者操作（只有当前 P 会从头部弹出），无锁。
优先从头部取是为了时间局部性（temporal locality）——最近放入的对象更可能还在 CPU 缓存中。

## 4.3 第三步：慢路径 getSlow

```go
func (p *Pool) getSlow(pid int) any {
    size := runtime_LoadAcquintptr(&p.localSize)
    locals := p.local
    // 尝试从其他 P 的 shared 队列尾部偷取
    for i := 0; i < int(size); i++ {
        l := indexLocal(locals, (pid+i+1)%int(size))
        if x, _ := l.shared.popTail(); x != nil {
            return x
        }
    }

    // 尝试从 victim cache 中获取
    size = atomic.LoadUintptr(&p.victimSize)
    if uintptr(pid) >= size {
        return nil
    }
    locals = p.victim
    l := indexLocal(locals, pid)
    if x := l.private; x != nil {
        l.private = nil
        return x
    }
    for i := 0; i < int(size); i++ {
        l := indexLocal(locals, (pid+i)%int(size))
        if x, _ := l.shared.popTail(); x != nil {
            return x
        }
    }

    // victim 也空了，标记为空避免后续无效遍历
    atomic.StoreUintptr(&p.victimSize, 0)
    return nil
}
```

getSlow 的查找顺序：

> 1 偷取其他 P：遍历所有其他 P 的 shared 队列，从尾部 popTail（CAS 操作，多消费者安全）。
这就是 work-stealing 策略，避免某些 P 的对象堆积而其他 P 缺乏对象。

> 2 victim cache 的 private：从上一轮 GC 留存的 victim 缓存中查找。

> 3 victim cache 的 shared：遍历 victim 中所有 P 的 shared 队列。

> 4 标记 victim 为空：如果 victim 中也找不到，将 victimSize 设为 0，后续 Get 不再尝试。

## 4.4 第四步：调用 New

如果所有缓存层都未命中，且用户设置了 New 函数，则调用 New 创建新对象。
如果 New 也未设置，返回 nil。


# 5 pin 机制：绑定 P 与禁止抢占

```go
func (p *Pool) pin() (*poolLocal, int) {
    if p == nil {
        panic("nil Pool")
    }
    pid := runtime_procPin()    // 禁止当前 goroutine 被抢占
    s := runtime_LoadAcquintptr(&p.localSize)
    l := p.local
    if uintptr(pid) < s {
        return indexLocal(l, pid), pid  // 快路径：直接索引
    }
    return p.pinSlow()                  // 慢路径：需要扩容
}
```

pin 的作用：

> 1 禁止抢占：runtime_procPin() 将当前 goroutine 绑定到当前 P，确保在操作 poolLocal 期间不会被调度到其他 P。
这是 Pool 无锁设计的基础——如果 goroutine 在操作 private 时被抢占到另一个 P，就会出现竞争。

> 2 快路径索引：如果 local 数组已初始化且 pid 在范围内，直接通过指针运算索引到对应的 poolLocal。

> 3 慢路径扩容（pinSlow）：当 GOMAXPROCS 变化或首次使用时，需要重新分配 local 数组。
pinSlow 中会先 unpin → 加全局锁 → 重新 pin → 分配新数组，并将 Pool 注册到全局 allPools 列表中。


# 6 GC 清理与 victim cache

## 6.1 poolCleanup 函数

```go
func poolCleanup() {
    // 此函数在 GC 开始时被调用（STW 阶段）

    // 清除上一轮的 victim cache
    for _, p := range oldPools {
        p.victim = nil
        p.victimSize = 0
    }

    // 将当前的 local 转移到 victim
    for _, p := range allPools {
        p.victim = p.local
        p.victimSize = p.localSize
        p.local = nil
        p.localSize = 0
    }

    oldPools, allPools = allPools, nil
}
```

poolCleanup 通过 `runtime_registerPoolCleanup` 注册，在每次 GC 的 STW 阶段被调用。

## 6.2 为什么不直接清空？

如果每次 GC 直接清空所有缓存对象，会导致 GC 后瞬间大量 New 调用，产生分配风暴。

victim cache 机制借鉴了 CPU 缓存的设计思想，提供了一个缓冲期：

| GC 周期 | local | victim | 效果 |
|---------|-------|--------|------|
| GC 前 | 有对象 | 上一轮的对象 | 正常使用 |
| 第 N 次 GC | 清空 → nil | local 转入 victim | 对象还有一次被复用的机会 |
| 第 N+1 次 GC | 新积累的对象 | 清空（真正回收） | 两轮未被使用的对象才被回收 |

**关键洞察**：一个对象需要经历**两个完整的 GC 周期**都未被使用，才会被真正回收。
这有效平滑了 GC 清理对应用性能的冲击。

## 6.3 对象生命周期

```
Put(obj) → private 或 shared
                ↓
          第 N 次 GC
                ↓
        local → victim（降级，仍可被 Get）
                ↓
         第 N+1 次 GC
                ↓
     victim → nil（如果两轮未被使用，真正回收）
```


# 7 无锁队列：poolDequeue 与 poolChain

## 7.1 poolDequeue：固定大小的无锁环形队列

```go
type poolDequeue struct {
    headTail atomic.Uint64 // 高 32 位 = head，低 32 位 = tail
    vals     []eface       // 环形缓冲区，大小必须是 2 的幂
}
```

poolDequeue 是一个**单生产者多消费者（SPMC）**的无锁队列：
- 生产者（当前 P）：可以从头部 pushHead 和 popHead。
- 消费者（其他 P）：只能从尾部 popTail。

headTail 将 head 和 tail 打包在一个 uint64 中，通过原子操作实现无锁并发：
```go
func (d *poolDequeue) unpack(ptrs uint64) (head, tail uint32) {
    head = uint32((ptrs >> 32) & mask)
    tail = uint32(ptrs & mask)
    return
}
```

**pushHead**（生产者独占，无需 CAS）：
检查队列是否已满 → 写入值 → 原子递增 head。

**popHead**（生产者独占）：
CAS 递减 head → 取出值 → 清零槽位。

**popTail**（多消费者竞争）：
CAS 递增 tail → 取出值 → 原子清零 typ 字段（通知 pushHead 槽位已释放）。

## 7.2 poolChain：动态增长的队列链

```go
type poolChain struct {
    head *poolChainElt              // 生产者推入端
    tail atomic.Pointer[poolChainElt] // 消费者弹出端
}

type poolChainElt struct {
    poolDequeue
    next, prev atomic.Pointer[poolChainElt]
}
```

poolChain 是 poolDequeue 的动态版本，由一个双向链表串联多个 poolDequeue：
- 初始大小为 8，每次扩容翻倍，最大为 dequeueLimit（2^30）。
- pushHead 时，如果当前 dequeue 满了，分配一个两倍大小的新 dequeue 追加到链表头部。
- popTail 时，从链表尾部的 dequeue 弹出，耗尽后移除该 dequeue 并前进到下一个。

```
tail                                           head
 ↓                                              ↓
[dequeue 8] ←→ [dequeue 16] ←→ [dequeue 32] ←→ [dequeue 64]
 (旧，可能已空)                                  (新，正在写入)
```

这种设计的优势：
- 初始开销小（8 个槽位），按需增长。
- 旧的 dequeue 耗尽后可以被 GC 回收，不会无限增长。
- 生产者和消费者在不同的 dequeue 上操作，减少竞争。


# 8 为什么采用 per-P 设计

## 8.1 假设使用全局锁

```go
// 假设的简单实现
type Pool struct {
    mu    sync.Mutex
    items []any
}

func (p *Pool) Get() any {
    p.mu.Lock()
    defer p.mu.Unlock()
    if len(p.items) > 0 {
        x := p.items[len(p.items)-1]
        p.items = p.items[:len(p.items)-1]
        return x
    }
    return nil
}
```

问题：
所有 goroutine 竞争同一把锁，高并发场景下锁竞争严重。
即使使用 RWMutex，Get 和 Put 都是写操作，无法利用读写分离。

## 8.2 per-P 设计的优势

| 操作 | 路径 | 同步方式 | 竞争程度 |
|------|------|---------|---------|
| Put → private | 当前 P | 直接赋值 | 零竞争 |
| Put → shared head | 当前 P | 无锁（单生产者） | 零竞争 |
| Get → private | 当前 P | 直接赋值 | 零竞争 |
| Get → shared head | 当前 P | 无锁（单消费者） | 零竞争 |
| Get → 其他 P shared tail | work-stealing | CAS | 低竞争 |

绝大多数操作（Put 和 Get 的热路径）都在当前 P 的 private 或 shared head 上完成，
完全无锁、无竞争。只有在本地缓存耗尽时才需要跨 P 偷取，此时使用 CAS 而非互斥锁，竞争程度极低。

这种设计与 Go 调度器的 per-P 运行队列（local run queue）思路一致：通过数据分区消除竞争。


# 9 使用注意事项

## 9.1 不要假设对象持久存在

```go
// 错误：把 Pool 当作持久缓存
var cache = sync.Pool{New: func() any { return loadFromDB() }}
// Pool 中的对象可能在下次 GC 时被清除！
```

Pool 中的对象可能在任意 GC 周期被清除，不适合做持久化缓存。

## 9.2 Put 前务必重置对象状态

```go
buf := pool.Get().(*bytes.Buffer)
defer func() {
    buf.Reset()    // 必须重置！否则下次 Get 到的 buffer 中有残留数据
    pool.Put(buf)
}()
```

## 9.3 避免在 Pool 中存放大对象

Pool 的对象在两个 GC 周期内不会被回收，如果存放大对象，可能导致内存占用过高。
对于大对象，考虑使用带大小限制的自定义 free list。

## 9.4 Pool 不能被复制

Pool 结构体中包含 noCopy 字段，`go vet` 会检测到 Pool 被复制的错误：
```go
var p1 sync.Pool
p2 := p1  // go vet: assignment copies lock value
```


# 10 总结

sync.Pool 的实现精妙地结合了多层缓存和无锁并发设计：

per-P 本地缓存：通过 pin 绑定 P + private 槽位，热路径上零竞争、零同步开销。
无锁双端队列：poolDequeue 实现单生产者多消费者的无锁环形队列，poolChain 提供动态扩容。
work-stealing：本地缓存耗尽时从其他 P 偷取，通过 CAS 操作避免锁竞争。
victim cache：GC 时不直接清空，而是降级到 victim，给对象一个额外的 GC 周期被复用的机会。

整体设计思路是：在绝大多数情况下（本地 Get/Put）做到零开销，只在少数情况下（跨 P 偷取、GC 清理）
付出有限的同步代价。这与 Go 运行时调度器的 per-P 架构一脉相承，体现了"数据分区消除竞争"的核心理念。
