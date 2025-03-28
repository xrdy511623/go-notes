
---
sync.Map源码分析
---

本文重点讲解sync.Map 是如何实现并发高性能操作的。

# 1 sync.Map 的底层数据结构

首先，我们来看下 sync.Map 的底层数据结构，它的核心是 read 和 dirty 两个 map 结构。read 存储了部分写入
Map 的内容，用来加速读操作。而 dirty 存储了全量内容，需要加锁才能读写数据。

```go
type Map struct {
    mu Mutex
    read atomic.Pointer[readOnly] // 无锁读map
    dirty map[any]*entry // 加锁读写map
    misses int
}

// readOnly is an immutable struct stored atomically in the Map.read field.
type readOnly struct {
    m       map[any]*entry
    amended bool // true if the dirty map contains some key not in m.
}
```


# 2 sync.Map的写入操作

接着，让我们来看下写入操作。当有 key-value 值写入时，如果这个 key 在 read 中不存在，接下来就要做新增操作，
它会加锁写入 dirty map 中，并且将 amended 标记设置为 true。而 amended 标记用于表示 dirty 中是否有不在
read 中的 key-value 值。这个操作过程你可以结合后面的示意图看一下，这样理解起来更直观。





![sync-map-write.png](images%2Fsync-map-write.png)





如果这个 key 在 read 中存在，则会进行更新操作，由于 read map 和 dirty map 里面存储的值是 entry 类型的指针，
且 entry 类型的成员变量也是 atomic.Pointer 类型（如后面代码所示）。


```go
// An entry is a slot in the map corresponding to a particular key.
type entry struct {
    p atomic.Pointer[any]
}
```

因此在更新时就像下面的图那样，可以直接用 CAS 无锁操作替换指针 p 指向的变量，而无需做加锁操作。





![sync-map-update.png](images%2Fsync-map-update.png)




# 3 sync.Map的读操作

然后，让我们来看看读取操作，我们还是结合具体代码来理解。

```go
// Load returns the value stored in the map for a key, or nil if no
// value is present.
// The ok result indicates whether value was found in the map.
func (m *Map) Load(key any) (value any, ok bool) {
    read := m.loadReadOnly()
    e, ok := read.m[key]
    if !ok && read.amended {
        m.mu.Lock()
        // Avoid reporting a spurious miss if m.dirty got promoted while we were
        // blocked on m.mu. (If further loads of the same key will not miss, it's
        // not worth copying the dirty map for this key.)
        read = m.loadReadOnly()
        e, ok = read.m[key]
        if !ok && read.amended {
            e, ok = m.dirty[key]
            // Regardless of whether the entry was present, record a miss: this key
            // will take the slow path until the dirty map is promoted to the read
            // map.
            m.missLocked()
        }
        m.mu.Unlock()
    }
    if !ok {
        return nil, false
    }
    return e.load()
}
```

当读取 key 对应的值时，会先从 read 中读取，当 read 中读不到，并且 amended 为 true 时，则会加锁从 dirty map 中读。
这里可能导致从 sync.Map 读取的性能劣化，因为它既要从 read 中读一遍，又要加锁从 dirty map 中读一遍。





![sync-map-read.png](images%2Fsync-map-read.png)





同时，每次 read 读不到，从 dirty map 中读时，它会调用 missLocked 方法，这个方法用于将 map 的 misses 字段加 1，
misses 字段用于表示 read 读未命中次数，如果 misses 值比较大，说明 read map 的数据可能比 dirty map 少了很多。
为了提升读性能，missLocked 方法里会将 dirty map 变成新的 read map，代码如下。


```go
func (m *Map) missLocked() {
    m.misses++
    if m.misses < len(m.dirty) {
        return
    }
    m.read.Store(&readOnly{m: m.dirty})
    m.dirty = nil
    m.misses = 0
}
```





![dirty-switch-read.png](images%2Fdirty-switch-read.png)





最后，让我们来看看另一个可能导致写入 sync.Map 的性能劣化的点。上面的 missLocked 方法，会将 dirty map 置为 nil，
当有新的 key-value 值写入时，为了能保持 dirty map 有全量数据，就像下面代码的 swap 方法，它会加锁并且调用
dirtyLocked 方法，遍历 read map 并全量赋值拷贝给 dirty map。你可以看看后面的代码，再想想这样写会不会有
什么问题？


```go
// Swap swaps the value for a key and returns the previous value if any.
// The loaded result reports whether the key was present.
func (m *Map) Swap(key, value any) (previous any, loaded bool) {
    ...
    m.mu.Lock()
    if !read.amended {
        // We're adding the first new key to the dirty map.
        // Make sure it is allocated and mark the read-only map as incomplete.
        m.dirtyLocked()
        m.read.Store(&readOnly{m: read.m, amended: true})
    }
    m.dirty[key] = newEntry(value)
    m.mu.Unlock()
    return previous, loaded
}

func (m *Map) dirtyLocked() {
    if m.dirty != nil {
        return
    }
    // read map全量复制到dirty
    read := m.loadReadOnly()
    m.dirty = make(map[any]*entry, len(read.m))
    for k, e := range read.m {
        if !e.tryExpungeLocked() {
            m.dirty[k] = e
        }
    }
}
```

不知道你有没有发现？当数据量比较大时，这样会导致大量数据的拷贝，性能会劣化严重。比如我们缓存几百万条数据，
就存在几百万条数据的赋值拷贝。通过上面 sync.Map 的原理分析，我们可以看出，sync.Map 是通过两个 map 来
实现读写分离，从而达到高性能读的目的。不过它存在下面几个缺点。由于有两个 map，因此占用内存会比较高。
更适用于读多写少的场景，当由于写比较多或者本地缓存没有全量数据时，会导致读 map 经常读不到数据，而
需要加锁再读一次，从而导致读性能退化。当数据量比较大时，如果写入触发读 map 向写 map 拷贝，
会导致较大的性能开销。可以看出来，sync.Map 的使用场景还是比较苛刻的。

那么在本地做大规模数据缓存时，我们是该选择分段锁实现的 map 还是 sync.Map 类型来缓存数据呢？答案是分段锁 map。
原因是我们很难准确地预估读写比例，而且读写比例也会随着业务的发展变化。此外，在大规模数据缓存时，两个 map 的内存
和拷贝开销也是不得不考虑的稳定性风险点，因此在大规模数据缓存时，我们一般使用分段锁实现的 map 来缓存数据。