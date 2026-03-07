
---
sync.Mutex互斥锁实现原理
---

# 1 互斥锁的基本概念
sync.Mutex 是 Go 标准库提供的一种互斥锁，用于保护共享资源，确保在任意时刻只有一个 goroutine 能访问临界区。它的核心方法包括：

Lock()：获取锁。如果锁已被其他 goroutine 持有，则当前 goroutine 会阻塞等待。
Unlock()：释放锁，并唤醒等待中的某个 goroutine。
sync.Mutex 的设计目标不仅是保证互斥性，还要兼顾性能（低开销）和公平性（避免 goroutine 长时间等待而无法获取锁）。

# 2 sync.Mutex 的内部结构

sync.Mutex 的实现依赖两个字段：

```go
type Mutex struct {
    state uint32  // 锁的状态
    sema  uint32  // 信号量，用于阻塞和唤醒 goroutine
}
```

## 2.1 state字段的位域设计
state 是一个 32 位无符号整数，通过位域管理多种状态：

```
 31                                 3    2        1       0
 ┌────────────────────────────────┬────┬────────┬───────┐
 │         waitersCount           │ S  │  W     │  L    │
 │          (29 bits)             │(1) │ (1)    │ (1)   │
 └────────────────────────────────┴────┴────────┴───────┘
  L = mutexLocked    锁定标志
  W = mutexWoken     唤醒标志
  S = mutexStarving  饥饿标志
```

对应源码中的常量定义：

```go
const (
    mutexLocked      = 1 << iota // 1, bit 0: 锁是否被持有
    mutexWoken                   // 2, bit 1: 是否有 goroutine 被唤醒
    mutexStarving                // 4, bit 2: 是否处于饥饿模式
    mutexWaiterShift = iota      // 3, waiter 计数从 bit 3 开始
)
```

- **locked**（bit 0）：表示锁是否被持有。0 表示未锁定，1 表示已锁定。
- **woken**（bit 1）：标记是否有一个 goroutine 被唤醒并正在尝试获取锁。该标志的作用是让 Unlock 知道有 goroutine 在自旋等待，**无需再通过信号量唤醒其他等待者**，减少无谓的上下文切换。
- **starving**（bit 2）：表示锁是否处于"饥饿模式"（后文详述）。
- **waiter**（bit 3-31, 29 bits）：记录等待锁的 goroutine 数量，最多支持 2^29 - 1 ≈ 5.4 亿个等待者。

这种位域设计将多个状态压缩到一个 uint32 变量中，既节省内存，又可以通过**单个原子操作同时读写多个状态**，提高了缓存效率。

## 2.2 sema字段
sema 是一个信号量，用于协调等待锁的 goroutine。当锁被竞争时，goroutine 会通过信号量进入阻塞状态；
锁释放时，信号量会唤醒等待者。


# 3  锁获取（Lock）的流程

sync.Mutex 的 Lock 方法分为快路径和慢路径，分别处理无竞争和竞争两种情况。

## 3.1 快路径：无竞争获取锁

```go
func (m *Mutex) Lock() {
    // 快路径：一次 CAS 尝试加锁
    if atomic.CompareAndSwapInt32(&m.state, 0, mutexLocked) {
        return
    }
    // 慢路径
    m.lockSlow()
}
```

通过原子操作 CAS 尝试将 state 从 0（未锁定）改为 mutexLocked（锁定）。如果成功，当前 goroutine 直接获取锁并返回；如果失败（说明锁已被占用或有其他状态位被设置），进入慢路径 `lockSlow()`。

快路径只需**一条原子指令**，没有函数调用、没有分支判断，适用于竞争不激烈的场景。这也是 Go 官方 benchmark 中 `BenchmarkMutexUncontended` 只需约 5ns 的原因。

## 3.2 慢路径：竞争获取锁

当锁被占用时，Lock 方法会执行以下步骤：

### 3.2.1 自旋（Spin）

在特定条件下，goroutine 会短暂自旋，循环检查锁是否释放：

```go
// lockSlow() 中的自旋逻辑（简化）
for {
    if old&(mutexLocked|mutexStarving) == mutexLocked && runtime_canSpin(iter) {
        // 设置 woken 标志，告知 Unlock 不必唤醒其他 goroutine
        if !awoke && old&mutexWoken == 0 && old>>mutexWaiterShift != 0 &&
            atomic.CompareAndSwapInt32(&m.state, old, old|mutexWoken) {
            awoke = true
        }
        runtime_doSpin() // 执行 PAUSE 指令（x86）
        iter++
        old = m.state
        continue
    }
    // ... 自旋失败，准备排队
}
```

**自旋的全部前置条件**（`runtime_canSpin` 的实现）：

1. 自旋次数 < 4（`active_spin = 4`）
2. 运行在多核处理器上（`GOMAXPROCS > 1`）
3. 当前 P 的本地运行队列为空（避免饿死其他 goroutine）
4. 锁未处于饥饿模式

每次自旋执行 30 次 `PAUSE` 指令（x86 架构），让出 CPU 流水线但不让出线程。

### 3.2.2 计算新状态并尝试 CAS

自旋失败后，goroutine 根据当前状态计算期望的新 state 值：

```go
new := old
// 正常模式下尝试加锁；饥饿模式下不设 locked 位（让给队首 waiter）
if old&mutexStarving == 0 {
    new |= mutexLocked
}
// 锁已被持有或处于饥饿模式，准备排队
if old&(mutexLocked|mutexStarving) != 0 {
    new += 1 << mutexWaiterShift // waiter 计数 +1
}
// 当前 goroutine 等待超过 1ms 且锁仍被持有，切换为饥饿模式
if starving && old&mutexLocked != 0 {
    new |= mutexStarving
}
if awoke {
    new &^= mutexWoken // 清除 woken 标志
}
```

### 3.2.3 加入等待队列

如果 CAS 成功但未获得锁，goroutine 通过信号量（`runtime_SemacquireMutex`）进入阻塞状态：

```go
// queueLifo: 首次排队放队尾（FIFO），被唤醒后重新排队放队首（LIFO）
runtime_SemacquireMutex(&m.sema, queueLifo, 1)
```

关键细节：**首次等待的 goroutine 加入队尾**，但**被唤醒后未抢到锁、重新排队的 goroutine 加入队首**。这保证了被唤醒的 goroutine 在下次释放锁时优先被再次唤醒。

### 3.2.4 唤醒与重试

被唤醒的 goroutine 检查自己的等待时间：
- 如果等待超过 1ms → 设置 `starving = true`，在下一轮 CAS 中切换锁为饥饿模式
- 如果锁已处于饥饿模式 → 被唤醒的 goroutine **直接获得锁**（此时新来的 goroutine 不会抢占）
- 如果是正常模式 → 被唤醒的 goroutine 需要**与新来的 goroutine 竞争**，往往处于劣势（新 goroutine 已在 CPU 上运行，cache 热）


# 4 锁释放（Unlock）的流程

## 4.1 快路径释放

```go
func (m *Mutex) Unlock() {
    // 快路径：清除 locked 位
    new := atomic.AddInt32(&m.state, -mutexLocked)
    if new != 0 {
        // 有其他状态位被设置（有等待者、woken、starving），进入慢路径
        m.unlockSlow(new)
    }
}
```

如果减去 mutexLocked 后 state 为 0（无等待者、无唤醒标志、非饥饿模式），直接返回。

## 4.2 Unlock 对未加锁 Mutex 的 panic

```go
func (m *Mutex) unlockSlow(new int32) {
    if (new+mutexLocked)&mutexLocked == 0 {
        fatal("sync: unlock of unlocked mutex")
    }
    // ...
}
```

**对一个未加锁的 Mutex 调用 Unlock 会直接 panic**（`fatal` 不可被 `recover` 捕获）。这是一个常见的编程错误，通常发生在：
- 多次调用 Unlock
- 在未持有锁的 goroutine 中调用 Unlock
- defer Unlock 后又手动 Unlock

## 4.3 慢路径：唤醒等待者

**正常模式**：递减 waiter 计数，通过 `runtime_Semrelease` 唤醒一个等待者。但如果已有 goroutine 被唤醒（woken 位已设置）或锁已被新 goroutine 抢占，则不再重复唤醒。

**饥饿模式**：直接将锁**移交**给等待队列队首的 goroutine（`handoff = true`），被唤醒的 goroutine 无需竞争直接获得锁。

```go
if old>>mutexWaiterShift == 1 || old&(mutexLocked|mutexWoken|mutexStarving) != 0 {
    return // 没有需要唤醒的等待者，或者已经有 goroutine 在处理
}
// 递减 waiter 计数并设置 woken 标志
new = (old - 1<<mutexWaiterShift) | mutexWoken
if atomic.CompareAndSwapInt32(&m.state, old, new) {
    runtime_Semrelease(&m.sema, false, 1) // 正常模式
    return
}
```


# 5 公平性机制：饥饿模式

为了避免某些 goroutine 因长时间等待而“饥饿”，sync.Mutex 引入了饥饿模式（starving mode）。

## 5.1 触发条件
当一个 goroutine 等待锁的时间超过 1 毫秒，锁会进入饥饿模式。
这通过内部时间检查实现，表明竞争激烈且存在潜在的不公平。

## 5.2 饥饿模式的行为

- **禁止自旋**：新来的 goroutine 不再自旋，直接加入等待队列尾部。
- **禁止抢占**：新来的 goroutine 不会尝试设置 locked 位，锁的所有权**直接移交**给队首等待者。
- **FIFO 调度**：锁释放时，通过 `runtime_Semrelease(&m.sema, true, 1)` 中的 `handoff=true` 参数，将信号量**直接交给队首 goroutine**，而非广播唤醒。

## 5.3 退出饥饿模式的条件

被唤醒的 goroutine 获得锁后，会检查是否应退出饥饿模式，满足**任一**条件即退出：

1. **该 goroutine 的等待时间 < 1ms**（说明竞争已不激烈）
2. **它是最后一个等待者**（waiter 计数为 0，队列已空）

```go
// 被唤醒后，重新计算自己的等待时长
starving = starving || runtime_nanotime()-waitStartTime > starvationThresholdNs
old = m.state

// 锁处于饥饿模式 → 被唤醒的 goroutine 直接获得锁
if old&mutexStarving != 0 {
    delta := int32(mutexLocked - 1<<mutexWaiterShift) // 设置 locked + waiter 计数 -1
    // 退出饥饿模式的判断：自己等待 < 1ms，或自己是最后一个 waiter
    if !starving || old>>mutexWaiterShift == 1 {
        delta -= mutexStarving // 清除饥饿标志
    }
    atomic.AddInt32(&m.state, delta)
    break // 获得锁，退出 for 循环
}
```

饥饿模式将锁的获取从"抢占式"变为"排队式"，显著提升公平性。根据 Go 官方的 benchmark，饥饿模式将尾延迟（tail latency）从秒级降低到毫秒级。

# 6 性能优化手段
sync.Mutex 在实现中采用多种手段优化性能：

## 6.1 快路径无锁操作
无竞争时，通过单个原子 CAS 操作获取锁，减少开销。

## 6.2 自旋机制
在锁竞争不激烈时，自旋避免阻塞，提升吞吐量。
自旋次数受限，避免过度消耗 CPU。

## 6.3 批量唤醒控制
锁释放时，仅唤醒一个 goroutine，避免“惊群效应”（多 goroutine 同时竞争锁）。

## 6.4 状态压缩
使用位域将锁状态压缩到 uint32，减少内存占用和缓存压力。

# 7 性能与公平性的平衡

sync.Mutex 通过动态机制在性能和公平性之间找到平衡：

| 维度 | 正常模式 | 饥饿模式 |
|------|---------|---------|
| 设计目标 | 高吞吐量 | 公平性、低尾延迟 |
| 新 goroutine | 可自旋、可抢占 | 直接排队尾，不自旋 |
| 锁释放 | 唤醒队首 waiter，需与新 goroutine 竞争 | 锁直接移交给队首 waiter（handoff） |
| 被唤醒者 | 劣势方（cache 冷、需重新调度） | 保证获得锁 |
| 切换条件 | waiter 等待 < 1ms 或队列空 | waiter 等待 ≥ 1ms |

动态切换根据等待时间自动调整模式。在竞争激烈时倾向公平性，竞争较弱时优化性能。

# 8 sync.RWMutex 的实现原理

sync.RWMutex 在 Mutex 基础上实现了读写分离，允许多个读操作并发执行，但写操作互斥。

## 8.1 内部结构

```go
type RWMutex struct {
    w           Mutex        // 写锁复用 Mutex
    writerSem   uint32       // 写等待者的信号量
    readerSem   uint32       // 读等待者的信号量
    readerCount atomic.Int32 // 当前活跃读者数量（可为负数）
    readerWait  atomic.Int32 // 写者需要等待完成的读者数量
}

const rwmutexMaxReaders = 1 << 30 // 最大读者数
```

## 8.2 读锁（RLock / RUnlock）

```go
func (rw *RWMutex) RLock() {
    // 原子递增 readerCount
    if rw.readerCount.Add(1) < 0 {
        // readerCount 为负数，说明有写者在等待，阻塞在 readerSem
        runtime_SemacquireRWMutexR(&rw.readerSem, false, 0)
    }
}
```

- 无写者时：`readerCount.Add(1)` 返回正数，直接获得读锁，**无需任何锁操作**。
- 有写者等待时：`readerCount` 为负数（见下文），新读者阻塞在 `readerSem`。

## 8.3 写锁（Lock / Unlock）—— readerCount 反转技巧

```go
func (rw *RWMutex) Lock() {
    rw.w.Lock() // 先获取底层 Mutex，阻止其他写者
    // 将 readerCount 减去 rwmutexMaxReaders，使其变为负数
    // 这是"写者到来"的信号
    r := rw.readerCount.Add(-rwmutexMaxReaders) + rwmutexMaxReaders
    // r 是反转前的实际读者数量
    if r != 0 && rw.readerWait.Add(r) != 0 {
        // 还有活跃读者，写者阻塞等待它们完成
        runtime_SemacquireRWMutex(&rw.writerSem, false, 0)
    }
}
```

**核心设计**：写者通过 `readerCount -= rwmutexMaxReaders` 将 readerCount 反转为负数。这一操作同时达成两个目标：
1. **阻止新读者**：新读者调用 `RLock` 时看到 `readerCount < 0`，知道有写者在等待，主动阻塞
2. **统计存量读者**：反转前的值 `r` 就是当前活跃的读者数量，写者等待它们全部释放

写者释放锁时恢复 `readerCount`：

```go
func (rw *RWMutex) Unlock() {
    // 将 readerCount 加回 rwmutexMaxReaders，恢复为正数
    r := rw.readerCount.Add(rwmutexMaxReaders)
    // 唤醒所有被阻塞的读者
    for i := 0; i < int(r); i++ {
        runtime_Semrelease(&rw.readerSem, false, 0)
    }
    rw.w.Unlock() // 释放底层 Mutex
}
```

## 8.4 写者优先的实现

当写者到来后，readerCount 变为负数，**后续所有新读者都会阻塞**。这意味着写者不会被源源不断的新读者饿死。已经持有读锁的 goroutine 不受影响，可以正常完成并 RUnlock。

| 场景 | readerCount 值 | 行为 |
|------|---------------|------|
| 无写者，5 个读者 | +5 | 新读者可直接获取读锁 |
| 写者到来 | 5 - 2^30 (负数) | 新读者阻塞，存量读者继续 |
| 存量读者全部释放 | -2^30 (仍为负数) | 写者获得写锁 |
| 写者释放 | 0 + 2^30 → 恢复 | 被阻塞的读者全部唤醒 |


# 9 总结

sync.Mutex 的实现原理体现了 Go 对并发控制的精妙设计：

- **内部结构**：通过位域（state 的 4 个逻辑字段）和信号量高效管理锁状态和等待队列。
- **获取流程**：快路径一条 CAS 指令加锁，慢路径通过自旋（≤4 次）和信号量阻塞处理竞争。
- **公平性**：饥饿模式（等待 > 1ms 触发）确保先来先服务，将尾延迟从秒级降至毫秒级。
- **性能**：原子操作、自旋、状态压缩、woken 标志避免惊群，多层次降低开销。
- **Unlock 安全**：对未加锁的 Mutex 调用 Unlock 会 fatal panic，不可 recover。
- **RWMutex**：通过 readerCount 反转技巧（减去 2^30）实现写者优先，在无写者时读锁操作零开销。

sync.Mutex 在高并发场景下，既保证了互斥性，又在性能与公平性之间取得了出色平衡，是 Go 并发原语的典范。

