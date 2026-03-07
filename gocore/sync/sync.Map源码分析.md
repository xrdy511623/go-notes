
---
sync.Map源码分析
---

本文重点讲解sync.Map 如何通过 read/dirty 双 map 读写分离、entry 指针的三态设计和 dirty 提升机制，在读多写少场景下实现高性能的并发访问。


# 1 sync.Map的设计目标

sync.Map 是 Go 标准库提供的并发安全 map，针对两种特定场景做了优化：
1. 键值对只写入一次但被多次读取（如只增长的缓存）。
2. 多个 goroutine 读写互不重叠的键集合。

在这两种场景下，sync.Map 可以显著减少锁竞争，性能优于 map + Mutex/RWMutex 方案。

其核心设计思路：
读写分离：read map 提供无锁读，dirty map 承接写入，两者通过 entry 指针共享数据。
延迟删除：删除操作不真正移除键，而是将 entry 标记为 nil/expunged，避免加锁。
自动提升：当 read 未命中次数累积到阈值时，dirty 整体提升为新的 read。


# 2 sync.Map的源码结构

## 2.1 核心数据结构

```go
type Map struct {
    mu Mutex
    read atomic.Pointer[readOnly] // 无锁读 map，原子存取
    dirty map[any]*entry           // 加锁读写 map，包含全量非 expunged 数据
    misses int                     // read 未命中计数
}

type readOnly struct {
    m       map[any]*entry
    amended bool // true 表示 dirty 中有 read 中不存在的 key
}
```

关键字段：
read：原子指针，指向不可变的 readOnly 结构。读操作无需加锁，直接原子加载。
dirty：普通 map，所有读写都需要持有 mu。包含全量非 expunged 的数据。
misses：记录 read 未命中次数。当 misses >= len(dirty) 时，dirty 提升为新的 read。
amended：标记 dirty 中是否有 read 中不存在的 key。如果 amended 为 false，read 中就有全量数据，无需查 dirty。

## 2.2 entry：指针的三态设计

```go
var expunged = new(any) // 全局哨兵指针

type entry struct {
    p atomic.Pointer[any]
}
```

entry.p 有三种状态，这是 sync.Map 最精妙的设计：

| 状态 | p 的值 | 含义 |
|------|--------|------|
| 正常 | 指向实际值 | 键值对有效，read 和 dirty 都可见 |
| 软删除 | nil | 键已被 Delete，但 entry 仍在 read（可能也在 dirty）中 |
| 硬删除 | expunged | 键已被删除，且 entry 不在 dirty 中（只在 read 中残留） |

**为什么需要区分 nil 和 expunged？**

当 dirty 为 nil，需要创建新的 dirty 时（dirtyLocked），要从 read 全量复制。
此时需要跳过已删除的键——但如果只用 nil 标记删除，就无法区分"已删除但在 dirty 中"和
"已删除且不在 dirty 中"这两种状态。

expunged 就是为了这个区分而设计的：
- nil → 已删除，但如果 dirty 存在，dirty 中也有这个 entry（共享指针）。
- expunged → 已删除，且不在 dirty 中。下次 Store 这个 key 时需要先加回 dirty。

```
entry 状态转换：

  正常值 ←→ nil（Delete/Store，CAS 操作）
    ↓
  nil → expunged（dirtyLocked 创建新 dirty 时，CAS 转换）
    ↓
  expunged → nil → 正常值（Store 时 unexpunge + 加回 dirty）
```


# 3 Load 的实现

```go
func (m *Map) Load(key any) (value any, ok bool) {
    read := m.loadReadOnly()
    e, ok := read.m[key]
    if !ok && read.amended {
        m.mu.Lock()
        read = m.loadReadOnly()          // 双重检查
        e, ok = read.m[key]
        if !ok && read.amended {
            e, ok = m.dirty[key]
            m.missLocked()               // 记录未命中
        }
        m.mu.Unlock()
    }
    if !ok {
        return nil, false
    }
    return e.load()
}
```

## 3.1 快路径：read 命中

```go
read := m.loadReadOnly()
e, ok := read.m[key]
```

原子加载 read，直接在 read.m 中查找。如果命中，调用 e.load() 返回值。
整个过程无锁、无 CAS，仅一次原子 Load，这是最快的路径。

## 3.2 慢路径：read 未命中

```go
if !ok && read.amended {
    m.mu.Lock()
    read = m.loadReadOnly()    // 再次检查（double-check）
    e, ok = read.m[key]
    if !ok && read.amended {
        e, ok = m.dirty[key]
        m.missLocked()
    }
    m.mu.Unlock()
}
```

当 read 中找不到且 amended 为 true（dirty 中可能有），才加锁查 dirty。

**双重检查**：加锁后再检查一次 read，因为在等待锁的期间，dirty 可能已被提升为新的 read。

**missLocked**：无论 dirty 中是否找到，都记录一次 miss。这确保即使反复查询不存在的 key，
也能触发 dirty 提升，避免永远走慢路径。

![sync-map-read.png](images%2Fsync-map-read.png)


# 4 Store/Swap 的实现

Store 是 Swap 的封装，核心逻辑在 Swap 中：

```go
func (m *Map) Swap(key, value any) (previous any, loaded bool) {
    read := m.loadReadOnly()
    if e, ok := read.m[key]; ok {
        if v, ok := e.trySwap(&value); ok {   // 快路径：CAS 直接更新
            if v == nil {
                return nil, false
            }
            return *v, true
        }
    }

    m.mu.Lock()
    read = m.loadReadOnly()
    if e, ok := read.m[key]; ok {
        if e.unexpungeLocked() {              // key 之前被 expunged
            m.dirty[key] = e                  // 加回 dirty
        }
        if v := e.swapLocked(&value); v != nil {
            loaded = true
            previous = *v
        }
    } else if e, ok := m.dirty[key]; ok {    // key 只在 dirty 中
        if v := e.swapLocked(&value); v != nil {
            loaded = true
            previous = *v
        }
    } else {                                  // 全新的 key
        if !read.amended {
            m.dirtyLocked()                   // 首次写入 dirty，从 read 复制
            m.read.Store(&readOnly{m: read.m, amended: true})
        }
        m.dirty[key] = newEntry(value)
    }
    m.mu.Unlock()
    return previous, loaded
}
```

## 4.1 快路径：key 在 read 中且未被 expunged

```go
if e, ok := read.m[key]; ok {
    if v, ok := e.trySwap(&value); ok {
        // ...
    }
}
```

由于 read 和 dirty 共享同一个 entry 指针，CAS 更新 entry.p 后，
read 和 dirty 都能看到新值，无需加锁。

![sync-map-update.png](images%2Fsync-map-update.png)

## 4.2 慢路径：三种情况

加锁后的处理分三种情况：

**情况一：key 在 read 中但被 expunged**
```go
if e.unexpungeLocked() {
    m.dirty[key] = e    // 将 entry 加回 dirty
}
e.swapLocked(&value)    // 更新值
```
先 CAS 将 expunged 改回 nil（unexpunge），再把 entry 加到 dirty 中，最后更新值。

**情况二：key 只在 dirty 中**
直接在 dirty 的 entry 上更新值。

**情况三：全新的 key**
```go
if !read.amended {
    m.dirtyLocked()    // 从 read 复制创建 dirty（如果 dirty 为 nil）
    m.read.Store(&readOnly{m: read.m, amended: true})
}
m.dirty[key] = newEntry(value)
```
如果 dirty 还未创建，先从 read 全量复制（跳过 expunged 的 entry）；
然后将新 entry 写入 dirty，并设置 amended = true。

![sync-map-write.png](images%2Fsync-map-write.png)


# 5 Delete 的实现

Delete 是 LoadAndDelete 的封装：

```go
func (m *Map) LoadAndDelete(key any) (value any, loaded bool) {
    read := m.loadReadOnly()
    e, ok := read.m[key]
    if !ok && read.amended {
        m.mu.Lock()
        read = m.loadReadOnly()
        e, ok = read.m[key]
        if !ok && read.amended {
            e, ok = m.dirty[key]
            delete(m.dirty, key)   // 从 dirty 中真正删除
            m.missLocked()
        }
        m.mu.Unlock()
    }
    if ok {
        return e.delete()          // CAS 将 entry.p 置为 nil（软删除）
    }
    return nil, false
}

func (e *entry) delete() (value any, ok bool) {
    for {
        p := e.p.Load()
        if p == nil || p == expunged {
            return nil, false
        }
        if e.p.CompareAndSwap(p, nil) {   // CAS 置 nil，不移除 entry
            return *p, true
        }
    }
}
```

**删除策略**：
- key 在 read 中：CAS 将 entry.p 置为 nil（软删除）。entry 本身不从 read.m 中移除，
  因为 read.m 是不可变的，不能直接 delete。
- key 只在 dirty 中：直接 delete(m.dirty, key) 真正删除。


# 6 dirty 的生命周期

dirty 的创建、提升和销毁构成了 sync.Map 的核心循环：

## 6.1 创建：dirtyLocked

```go
func (m *Map) dirtyLocked() {
    if m.dirty != nil {
        return
    }
    read := m.loadReadOnly()
    m.dirty = make(map[any]*entry, len(read.m))
    for k, e := range read.m {
        if !e.tryExpungeLocked() {   // nil → expunged，跳过已删除的
            m.dirty[k] = e           // 共享 entry 指针，不复制值
        }
    }
}

func (e *entry) tryExpungeLocked() (isExpunged bool) {
    p := e.p.Load()
    for p == nil {
        if e.p.CompareAndSwap(nil, expunged) {
            return true             // nil → expunged，标记为硬删除
        }
        p = e.p.Load()
    }
    return p == expunged
}
```

创建新 dirty 时遍历 read，将 entry.p == nil 的项 CAS 转换为 expunged 并跳过，
其余项共享 entry 指针复制到 dirty。

**注意**：这里复制的是 entry 指针，不是值本身，所以 read 和 dirty 中的同一个 key 指向
同一个 entry，更新时只需 CAS 替换 entry.p 就能同时对两边生效。

## 6.2 提升：missLocked

```go
func (m *Map) missLocked() {
    m.misses++
    if m.misses < len(m.dirty) {
        return
    }
    m.read.Store(&readOnly{m: m.dirty})  // dirty 整体变为新 read
    m.dirty = nil
    m.misses = 0
}
```

当 misses 达到 dirty 的长度时，将 dirty 直接提升为新的 read。
提升操作是 O(1)——只是原子替换一个指针，不涉及数据复制。

![dirty-switch-read.png](images%2Fdirty-switch-read.png)

## 6.3 完整循环

```
初始状态：read 有数据，dirty = nil，amended = false
      ↓
新 key 写入 → dirtyLocked() 从 read 复制创建 dirty → amended = true
      ↓
持续写入新 key → 写入 dirty
      ↓
持续 Load 未命中 → misses 累加
      ↓
misses >= len(dirty) → dirty 提升为新 read → dirty = nil → amended = false
      ↓
回到初始状态，等待下一轮写入
```


# 7 Range 的实现

```go
func (m *Map) Range(f func(key, value any) bool) {
    read := m.loadReadOnly()
    if read.amended {
        m.mu.Lock()
        read = m.loadReadOnly()
        if read.amended {
            read = readOnly{m: m.dirty}
            copyRead := read
            m.read.Store(&copyRead)   // 提前触发 dirty → read 提升
            m.dirty = nil
            m.misses = 0
        }
        m.mu.Unlock()
    }
    for k, e := range read.m {
        v, ok := e.load()
        if !ok {
            continue                  // 跳过已删除的 entry
        }
        if !f(k, v) {
            break
        }
    }
}
```

Range 本身就是 O(N) 操作，因此在开始遍历前直接将 dirty 提升为 read，
摊销了这次复制的开销。提升后遍历 read.m 即可，无需持有锁。

**注意**：Range 不保证一致性快照——遍历期间的并发修改可能部分可见。


# 8 为什么需要 read/dirty 双 map

## 8.1 假设只用一个 map + RWMutex

```go
type Map struct {
    mu sync.RWMutex
    m  map[any]any
}
func (m *Map) Load(key any) (any, bool) {
    m.mu.RLock()
    v, ok := m.m[key]
    m.mu.RUnlock()
    return v, ok
}
```

问题：
RWMutex 的读锁在高并发下仍有开销（原子操作递增/递减 reader count）。
写操作会阻塞所有读操作，读多写少场景下写操作的影响被放大。

## 8.2 read/dirty 分离的优势

| 操作 | 路径 | 同步方式 | 竞争程度 |
|------|------|---------|---------|
| Load → read 命中 | 快路径 | 原子 Load | 零竞争 |
| Load → dirty 回退 | 慢路径 | Mutex | 有竞争 |
| Store → read 中已存在 | 快路径 | CAS | 低竞争 |
| Store → 新 key | 慢路径 | Mutex | 有竞争 |
| Delete → read 中存在 | 快路径 | CAS | 低竞争 |

在读多写少场景下，绝大多数操作命中 read 的快路径，完全无锁。
只有新增 key 或 read 未命中时才需要加锁，锁的持有时间也很短。


# 9 性能特性与适用场景

## 9.1 性能代价

**dirty 创建的 O(N) 开销**：当 dirty 为 nil 时写入新 key，需要遍历 read 全量复制。
如果 map 中有百万级条目，这次复制的开销不可忽视。

**双倍内存**：read 和 dirty 可能同时持有大量 entry（虽然共享 entry 指针，但 map 结构本身的
bucket 是各自独立的），内存占用高于单 map 方案。

**读性能退化**：如果写入大量新 key 导致 amended 长期为 true，read 命中率下降，
每次 Load 都需要加锁查 dirty，性能退化到比 RWMutex 更差（双重查找开销）。

## 9.2 适用场景

| 场景 | 是否适合 | 原因 |
|------|---------|------|
| 读多写少（缓存只增长） | 适合 | read 命中率高，接近无锁 |
| 各 goroutine 操作不同 key | 适合 | 减少锁竞争 |
| 读写比接近 1:1 | 不适合 | read 未命中频繁，加锁开销高 |
| 大量数据 + 频繁新增 key | 不适合 | dirty 创建的 O(N) 复制开销大 |
| 需要强一致性快照 | 不适合 | Range 不保证一致性 |

## 9.3 替代方案：分段锁

对于大规模数据缓存，分段锁（sharded map）通常是更稳健的选择：

```go
type ShardedMap struct {
    shards [256]struct {
        mu sync.RWMutex
        m  map[string]any
    }
}
func (sm *ShardedMap) getShard(key string) *shard {
    h := fnv32(key)
    return &sm.shards[h%256]
}
```

分段锁的优势：
- 性能不依赖读写比例，任何场景都稳定。
- 无 O(N) 全量复制的风险。
- 内存占用可控（只有一份数据）。
- 可以通过增加分段数来线性降低锁竞争。


# 10 总结

sync.Map 的实现通过 read/dirty 双 map 分离和 entry 三态指针设计，在特定场景下实现了接近无锁的读性能：

读写分离：read map 提供无锁读（原子 Load），dirty map 承接新写入（Mutex 保护）。
entry 三态：正常值/nil（软删除）/expunged（硬删除），用一个指针的三种状态编码了丰富的语义，
    使得删除操作无需加锁，dirty 创建时能正确跳过已删除的键。
指针共享：read 和 dirty 中同一个 key 指向同一个 entry，更新时 CAS 替换 entry.p 即可，
    无需同步两个 map。
自动提升：misses 累积到阈值后，dirty O(1) 提升为新 read，恢复读性能。

这套设计在读多写少场景下表现出色，但对写入模式敏感。理解 entry 的三态转换和 dirty 的生命周期，
是掌握 sync.Map 实现的关键。
