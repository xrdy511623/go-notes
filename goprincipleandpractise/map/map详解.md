
---
map详解
---

# 1 map的底层实现原理是什么？

**Go 语言的map采用的是哈希查找表，并且使用链表解决哈希冲突**

```golang
type hmap struct {
    // 元素个数，调用 len(map) 时，直接返回此值
   count     int
   flags     uint8
   // buckets 的对数 log_2
   B         uint8
   // overflow 的 bucket 近似数
   noverflow uint16
   // 计算 key 的哈希的时候会传入哈希函数
   hash0     uint32
    // 指向 buckets 数组，大小为 2^B
    // 如果元素个数为0，就为 nil
   buckets    unsafe.Pointer
   // 等量扩容的时候，buckets 长度和 oldbuckets 相等
   // 双倍扩容的时候，buckets 长度会是 oldbuckets 的两倍
   oldbuckets unsafe.Pointer
   // 指示扩容进度，小于此地址的 buckets 迁移完成
   nevacuate  uintptr
   extra *mapextra // optional fields
}
```

`B` 是 buckets 数组的长度的对数，也就是说 buckets 数组的长度就是 2^B。bucket 里面存储了 key 和 value。
buckets 是一个指针，最终它指向的是一个结构体：

```golang
type bmap struct {
    topbits  [8]uint8
    keys     [8]keytype
    values   [8]valuetype
    pad      uintptr
    overflow uintptr
}
```

`bmap` 就是我们常说的“桶”，桶里面会最多装 8 个 key，这些 key 之所以会落入同一个桶，是因为它们经过哈希计算后，哈希结果
是“一类”的。在桶内，又会根据 key 计算出来的 hash 值的高 8 位来决定 key 到底落入桶内的哪个位置（一个桶内最多有8个位置）。





![hmap-and-bmap.png](images%2Fhmap-and-bmap.png)





当 map 的 key 和 value 都不是指针，并且 size 都小于 128 字节的情况下，会把 bmap 标记为不含指针，这样可以避免 gc 时扫描整个 
hmap。但是，我们看 bmap 其实有一个 overflow 的字段，是指针类型的，破坏了 bmap 不含指针的设想，这时会把 overflow 移动到
extra 字段来。

```golang
type mapextra struct {
   // overflow[0] contains overflow buckets for hmap.buckets.
   // overflow[1] contains overflow buckets for hmap.oldbuckets.
   overflow [2]*[]*bmap

   // nextOverflow 包含空闲的 overflow bucket，这是预分配的 bucket
   nextOverflow *bmap
}
```

bmap 是存放 k-v键值对的地方，我们把视角拉近，仔细看 bmap 的内部组成。





![bmap-detail.png](images%2Fbmap-detail.png)





上图就是 bucket 的内存模型，`HOB Hash` 指的就是 top hash。 注意到 key 和 value 是各自放在一起的，并不是
`key/value/key/value/...` 这样的形式。源码里说明这样的好处是在某些情况下可以省略掉 padding 字段，节省内存空间。

如果按照 `key/value/key/value/...` 这样的模式存储，那在每一个 key/value 对之后都要额外 padding 7 个字节；
而将所有的 key，value 分别绑定到一起，这种形式 `key/key/.../value/value/...`，则只需要在最后添加 padding。

每个 bucket 设计成最多只能放 8 个 key-value 对，如果有第 9 个 key-value 落入当前的 bucket，那就需要再构建一个 
bucket ，通过 `overflow` 指针连接起来。





![make-map.png](images%2Fmake-map.png)





通过汇编语言可以看到，实际上底层调用的是 `makemap` 函数，主要做的工作就是初始化 `hmap` 结构体的各种字段，例如计算
B 的大小，设置哈希种子 hash0 等等。

```golang
func makemap(t *maptype, hint int, h *hmap) *hmap {
	mem, overflow := math.MulUintptr(uintptr(hint), t.bucket.size)
	if overflow || mem > maxAlloc {
		hint = 0
	}

	// 初始化hmap
	if h == nil {
		h = new(hmap)
	}
	h.hash0 = fastrand()
	
	// 计算B的大小
	B := uint8(0)
	for overLoadFactor(hint, B) {
		B++
	}
	h.B = B

	// 为初始化的hash table分配内存
	if h.B != 0 {
		var nextOverflow *bmap
		h.buckets, nextOverflow = makeBucketArray(t, h.B, nil)
		if nextOverflow != nil {
			h.extra = new(mapextra)
			h.extra.nextOverflow = nextOverflow
		}
	}

	return h
}

```

# 2 slice 和 map 分别作为函数参数时有什么区别？

注意，这个函数返回的结果：`*hmap`，它是一个指针，而`makeslice` 函数返回的是 `Slice` 结构体：
makemap 和 makeslice 的区别，带来一个不同点：当 map 和 slice 作为函数参数时，在函数参数内部对 map 的操作会
影响 map 自身；而对 slice 却不会。
主要原因：一个是指针（`*hmap`），一个是结构体（`slice`）。Go 语言中的函数传参都是值传递，在函数内部，参数会被
copy 到本地。`*hmap`指针 copy 完之后，仍然指向同一个 map，因此函数内部对 map 的操作会影响实参。而 slice 被
copy 后，会成为一个新的 slice，对它进行的操作不会影响到实参。

key 定位过程(哈希值低位找哈希桶，哈希值高位找在这个哈希桶的具体位置)
key 经过哈希计算后得到哈希值，共 64 个 bit 位（64位机，32位机就不讨论了，现在主流都是64位机），计算它到底要落在
哪个桶时，只会用到最后 B 个 bit 位。还记得前面提到过的 B 吗？如果 B = 5，那么桶的数量，也就是 buckets 数组的长度是
2^5 = 32。

例如，现在有一个 key 经过哈希函数计算后，得到的哈希结果是：

```shell
10010111 | 000011110110110010001111001010100010010110010101010 │ 01010
```

用最后的 5 个 bit 位，也就是 `01010`，值为 10，确定这个key所在的哈希桶，也就是 10 号桶。这个操作实际上就是取余操作，
但是取余开销太大，所以代码实现上用的位操作代替。
再用哈希值的高 8 位，找到此 key 在 bucket 中的位置，这是在寻找已有的 key。最开始桶内还没有 key，新加入的 key 会找到第
一个空位，放入。
buckets 编号就是桶编号，当两个不同的 key 落在同一个桶中，也就是发生了哈希冲突。冲突的解决手段是用链表法：在 
bucket 中，从前往后找到第一个空位。这样，在查找某个 key 时，先找到对应的桶，再去遍历 bucket 中的 key。





![hash-key-search.png](images%2Fhash-key-search.png)





上图中，假定 B = 5，所以 bucket 总数就是 2^5 = 32。首先计算出待查找 key 的哈希，使用低 5 位 `00110`，找到对应的 6 号 bucket，
使用高 8 位 `10010111`，对应十进制 151，在 6 号 bucket 中寻找 tophash 值（HOB hash）为 151 的 key，找到了 2 号槽位，
这样整个查找过程就结束了。
如果在 bucket 中没找到，并且 overflow 不为空，还要继续去 overflow bucket 中寻找，直到找到或是所有的 key 槽位都找遍了，
包括所有的 overflow bucket。

函数返回 h[key] 的指针，如果 h 中没有此 key，那就会返回一个 key 相应类型的零值，不会返回 nil。

说一下定位 key 和 value 的方法以及整个循环的写法。

```golang
// key 定位公式
k := add(unsafe.Pointer(b), dataOffset+i*uintptr(t.keysize))

// value 定位公式
v := add(unsafe.Pointer(b), dataOffset+bucketCnt*uintptr(t.keysize)+i*uintptr(t.valuesize))
```

b 是 bmap 的地址，这里 bmap 还是源码里定义的结构体，只包含一个 tophash 数组，经编译器扩充之后的结构体才包含 key，value，overflow 
这些字段。dataOffset 是 key 相对于 bmap 起始地址的偏移：

```golang
dataOffset = unsafe.Offsetof(struct {
      b bmap
      v int64
   }{}.v)
```
因此 bucket 里 key 的起始地址就是 unsafe.Pointer(b)+dataOffset。第 i 个 key 的地址就要在此基础上跨过 i 个 key 
的大小；而我们又知道，value 的地址是在所有 key 之后，因此第 i 个 value 的地址还需要加上所有 key 的偏移。理解了这些，
上面 key 和 value 的定位公式就很好理解了。

# 3 map的赋值过程

通过汇编语言可以看到，向 map 中插入或者修改 key，最终调用的是 `mapassign` 函数。
实际上插入或修改 key 的语法是一样的，只不过前者操作的 key 在 map 中不存在，而后者操作的 key 存在 map 中。
mapassign 有一个系列的函数，根据 key 类型的不同，编译器会将其优化为相应的“快速函数”。

```golang

|key 类型|插入|
|---|---|
|uint32|mapassign_fast32(t *maptype, h *hmap, key uint32) unsafe.Pointer|
|uint64|mapassign_fast64(t *maptype, h *hmap, key uint64) unsafe.Pointer|
|string|mapassign_faststr(t *maptype, h *hmap, ky string) unsafe.Pointer|
```

我们只用研究最一般的赋值函数 `mapassign`。

整体来看，流程非常简单：对 key 计算 hash 值，根据 hash 值按照之前的流程，找到要赋值的位置（可能是插入新 key，也可能是
更新老 key），对相应位置进行赋值。
源码大体和之前讲的类似，核心还是一个双层循环，外层遍历 bucket 和它的 overflow bucket，内层遍历整个 bucket 的各个 cell。

函数首先会检查 map 的标志位 flags。如果 flags 的写标志位此时被置 1 了，说明有其他协程在执行“写”操作，进而导致程序 panic。
这也说明了 map 对协程是不安全的。
我们知道扩容是渐进式的，如果 map 处在扩容的过程中，那么当 key 定位到了某个 bucket 后，需要确保这个 bucket 对应的老
bucket 完成了迁移过程。即老 bucket 里的 key 都要迁移到新的 bucket 中来（分裂到 2 个新 bucket），才能在新的 bucket 中
进行插入或者更新的操作。
上面说的操作是在函数靠前的位置进行的，只有进行完了这个搬迁操作后，我们才能放心地在新 bucket 里定位 key 要安置的地址，再进行之后的操作。
现在到了定位 key 应该放置的位置了，所谓找准自己的位置很重要。准备两个指针，一个（`inserti`）指向 key 的 hash 值在 tophash 数组所处
的位置，另一个(`insertk`)指向 cell 的位置（也就是 key 最终放置的地址），当然，对应 value 的位置就很容易定位出来了。这三者实际上都是关联的，
在 tophash 数组中的索引位置决定了 key 在整个 bucket 中的位置（共 8 个 key），而 value 的位置需要“跨过” 8 个 key 的长度。
在循环的过程中，inserti 和 insertk 分别指向第一个找到的空闲的 cell。如果之后在 map 没有找到 key 的存在，也就是说原来 map 中没有此 key，
这意味着插入新 key。那最终 key 的安置地址就是第一次发现的“空位”（tophash 是 empty）。
如果这个 bucket 的 8 个 key 都已经放置满了，那在跳出循环后，发现 inserti 和 insertk 都是空，这时候需要在 bucket 后面挂上
overflow bucket。当然，也有可能是在 overflow bucket 后面再挂上一个 overflow bucket。这就说明，太多 key hash 到了此 bucket。
在正式安置 key 之前，还要检查 map 的状态，看它是否需要进行扩容。如果满足扩容的条件，就主动触发一次扩容操作。
这之后，整个之前的查找定位 key 的过程，还得再重新走一次。因为扩容之后，key 的分布都发生了变化。
最后，会更新 map 相关的值，如果是插入新 key，map 的元素数量字段 count 值会加 1；在函数之初设置的 `hashWriting` 写标志出会清零。
另外，有一个重要的点要说一下。前面说的找到 key 的位置，进行赋值操作，实际上并不准确。我们看 `mapassign` 函数的原型就知道，函数并没有传入
value 值，所以赋值操作是什么时候执行的呢？

```golang

func mapassign(t *maptype, h *hmap, key unsafe.Pointer) unsafe.Pointer
```

答案还得从汇编语言中寻找。`mapassign` 函数返回的指针就是指向的 key 所对应的 value 值位置，有了地址，就很好操作赋值了。

# 4 map的删除过程

删除操作底层的执行函数是 `mapdelete`

```golang
func mapdelete(t *maptype, h *hmap, key unsafe.Pointer) 
```
当然，我们只关心 `mapdelete` 函数。它首先会检查 h.flags 标志，如果发现写标位是 1，直接 panic，因为这表明有其他协程同时
在进行写操作。
计算 key 的哈希，找到落入的 bucket。检查此 map 如果正在扩容的过程中，直接触发一次搬迁操作。
删除操作同样是两层循环，核心还是找到 key 的具体位置。寻找过程都是类似的，在 bucket 中挨个 cell 寻找。
找到对应位置后，对 key 或者 value 进行“清零”操作：

```golang
// 对 key 清零
if t.indirectkey {
   *(*unsafe.Pointer)(k) = nil
} else {
   typedmemclr(t.key, k)
}

// 对 value 清零
if t.indirectvalue {
   *(*unsafe.Pointer)(v) = nil
} else {
   typedmemclr(t.elem, v)
}
```
最后，将 count 值减 1，将对应位置的 tophash 值置成 `Empty`。

# 5 map的扩容过程

使用哈希表的目的就是要快速查找到目标 key，然而，随着向 map 中添加的 key 越来越多，key 发生碰撞的概率也越来越大。bucket 中的 8 个
cell 会被逐渐塞满，查找、插入、删除 key 的效率也会越来越低。最理想的情况是一个 bucket 只装一个 key，这样，就能达到 `O(1)` 的效率，
但这样空间消耗太大，用空间换时间的代价太高。
Go 语言采用一个 bucket 里装载 8 个 key，定位到某个 bucket 后，还需要再定位到具体的 key，这实际上又用了时间换空间。
当然，这样做，要有一个度，不然所有的 key 都落在了同一个 bucket 里，直接退化成了链表，各种操作的效率直接降为 O(n)，是不行的。
因此，需要有一个指标来衡量前面描述的情况，这就是负载因子。Go 源码里这样定义负载因子：

```shell
loadFactor := count / (2^B)
```
count 就是 map 的元素个数，2^B 表示 bucket 数量。
再来说触发 map 扩容的时机：在向 map 插入新 key 的时候，会进行条件检测，符合下面这 2 个条件，就会触发扩容：
1. 装载因子超过阈值，源码里定义的阈值是 6.5。
2. overflow 的 bucket 数量过多：当 B 小于 15，也就是 bucket 总数 2^B 小于 2^15 时，如果 overflow 的 bucket
3. 数量超过 2^B；当 B >= 15，也就是 bucket 总数 2^B 大于等于 2^15，如果 overflow 的 bucket 数量超过 2^15。

通过汇编语言可以找到赋值操作对应源码中的函数是 `mapassign`，对应扩容条件的源码如下：

```golang
// src/runtime/hashmap.go/mapassign
// 触发扩容时机
if !h.growing() && (overLoadFactor(int64(h.count), h.B) || tooManyOverflowBuckets(h.noverflow, h.B)) {
      hashGrow(t, h)
   }

// 装载因子超过 6.5
func overLoadFactor(count int64, B uint8) bool {
   return count >= bucketCnt && float32(count) >= loadFactor*float32((uint64(1)<<B))
}

// overflow buckets 太多
func tooManyOverflowBuckets(noverflow uint16, B uint8) bool {
   if B < 16 {
      return noverflow >= uint16(1)<<B
   }
   return noverflow >= 1<<15
}
```

解释一下：

第 1 点：我们知道，每个 bucket 有 8 个空位，在没有溢出，且所有的桶都装满了的情况下，装载因子算出来的结果是 8。因此当
装载因子超过 6.5 时，表明很多 bucket 都快要装满了，查找效率和插入效率都变低了。在这个时候进行扩容是有必要的。

第 2 点：是对第 1 点的补充。就是说在装载因子比较小的情况下，这时候 map 的查找和插入效率也很低，而第 1 点识别不出来这种情况。
表面现象就是计算装载因子的分子比较小，即 map 里元素总数少，但是 bucket 数量多（真实分配的 bucket 数量多，包括大量的 overflow bucket）。

不难想像造成这种情况的原因：不停地插入、删除元素。先插入很多元素，导致创建了很多 bucket，但是装载因子达不到第 1 点的临界值，未
触发扩容来缓解这种情况。之后，删除元素降低元素总数量，再插入很多元素，导致创建很多的 overflow bucket，但就是不会触犯第 1 点的
规定，你能拿我怎么办？overflow bucket 数量太多，导致 key 会很分散，查找插入效率低得吓人，因此出台第 2 点规定。这就像是一座空城，
房子很多，但是住户很少，都分散了，找起人来很困难。

对于命中条件 1，2 的限制，都会发生扩容。但是扩容的策略并不相同，毕竟两种条件应对的场景不同。

对于条件 1，元素太多，而 bucket 数量太少，很简单：将 B 加 1，bucket 最大数量（2^B）直接变成原来 bucket 数量的 2 倍。于是，
就有新老 bucket 了。注意，这时候元素都在老 bucket 里，还没迁移到新的 bucket 来。而且，新 bucket 只是最大数量变为原来最大数量（2^B）
的 2 倍（2^B * 2）。

对于条件 2，其实元素没那么多，但是 overflow bucket 数特别多，说明很多 bucket 都没装满。解决办法就是开辟一个新 bucket 空间，
将老 bucket 中的元素移动到新 bucket，使得同一个 bucket 中的 key 排列地更紧密。这样，原来，在 overflow bucket 中的 key 
可以移动到 bucket 中来。结果是节省空间，提高 bucket 利用率，map 的查找和插入效率自然就会提升。

对于条件 2 的解决方案，曹大的博客里还提出了一个极端的情况：如果插入 map 的 key 哈希都一样，就会落到同一个 bucket 里，超过 8 个
就会产生 overflow bucket，结果也会造成 overflow bucket 数过多。移动元素其实解决不了问题，因为这时整个哈希表已经退化成了一个链表，
操作效率变成了 `O(n)`。

再来看一下扩容具体是怎么做的。由于 map 扩容需要将原有的 key/value 重新搬迁到新的内存地址，如果有大量的 key/value 需要搬迁，
会非常影响性能。因此 Go map 的扩容采取了一种称为“渐进式”地方式，原有的 key 并不会一次性搬迁完毕，每次最多只会搬迁 2 个 bucket。

上面说的 `hashGrow()` 函数实际上并没有真正地“搬迁”，它只是分配好了新的 buckets，并将老的 buckets 挂到了 oldbuckets 字段上。
真正搬迁 buckets 的动作在 `growWork()` 函数中，而调用 `growWork()` 函数的动作是在 mapassign 和 mapdelete 函数中。也就
是插入或修改、删除 key 的时候，都会尝试进行搬迁 buckets 的工作。

搬迁的目的就是将老的 buckets 搬迁到新的 buckets。而通过前面的说明我们知道，应对条件 1，新的 buckets 数量是之前的一倍，应对条件 2，
新的 buckets 数量和之前相等。
对于条件 2，从老的 buckets 搬迁到新的 buckets，由于 buckets 数量不变，因此可以按序号来搬，比如原来在 0 号 bucket，到新的地方后，
仍然放在 0 号 bucket。
对于条件 1，就没这么简单了。要重新计算 key 的哈希，才能决定它到底落在哪个 bucket。例如，原来 B = 5，计算出 key 的哈希后，只用看它的低 5 位，
就能决定它落在哪个 bucket。扩容后，B 变成了 6，因此需要多看一位，它的低 6 位决定 key 落在哪个 bucket。这称为 `rehash`。





![rehash.png](images%2Frehash.png)





因此，某个 key 在搬迁前后 bucket 序号可能和原来相等，也可能是相比原来加上 2^B（原来的 B 值），取决于 hash 值 第 6 bit 位是 0  还是 1。
再明确一个问题：如果扩容后，B 增加了 1，意味着 buckets 总数是原来的 2 倍，原来 1 号的桶“裂变”到两个桶。
例如，原始 B = 2，1号 bucket 中有 2 个 key 的哈希值低 3 位分别为：010，110。由于原来 B = 2，所以低 2 位 `10` 决定它们落在 2 号桶，
现在 B 变成 3，所以 `010`、`110` 分别落入 2、6 号桶。

evacuate 函数每次只完成一个 bucket 的搬迁工作，因此要遍历完此 bucket 的所有的 cell，将有值的 cell copy 到新的地方。bucket 还会链接
overflow bucket，它们同样需要搬迁。因此会有 2 层循环，外层遍历 bucket 和 overflow bucket，内层遍历 bucket 的所有 cell。这样的循环在 
map 的源码里到处都是，要理解透了。


下面通过图来宏观地看一下扩容前后的变化。
扩容前，B = 2，共有 4 个 buckets，lowbits 表示 hash 值的低位。假设我们不关注其他 buckets 情况，专注在 2 号 bucket。并且假设 overflow 
太多，触发了等量扩容（对应于前面的条件 2）。

下面是扩容前:





![expand-before.png](images%2Fexpand-before.png)





扩容后:





![expand-after.png](images%2Fexpand-after.png)





扩容完成后，overflow bucket 消失了，key 都集中到了一个 bucket，更为紧凑了，提高了查找的效率。





![double-expand.png](images%2Fdouble-expand.png)





假设触发了 2 倍的扩容，那么扩容完成后，老 buckets 中的 key 分裂到了 2 个 新的 bucket。一个在 x part，一个在 y 的 part。依据是 hash 的 
lowbits。新 map 中 `0-3` 称为 x part，`4-7` 称为 y part。

# 6 map中的key 为什么是无序的

map 在扩容后，会发生 key 的搬迁，原来落在同一个 bucket 中的 key，搬迁后，有些 key 就要远走高飞了（bucket 序号加上了 2^B）。
而遍历的过程，就是按顺序遍历 bucket，同时按顺序遍历 bucket 中的 key。搬迁后，key 的位置发生了重大的变化，有些 key 飞上高枝，
有些 key 则原地不动。这样，遍历 map 的结果就不可能按原来的顺序了。
当然，如果我就一个 hard code 的 map，我也不会向 map 进行插入删除的操作，按理说每次遍历这样的 map 都会返回一个固定顺序的 key/value 
序列吧。的确是这样，但是 Go 杜绝了这种做法，因为这样会给新手程序员带来误解，以为这是一定会发生的事情，在某些情况下，可能会酿成大错。
当然，Go 做得更绝，当我们在遍历 map 时，并不是固定地从 0 号 bucket 开始遍历，每次都是从一个随机值序号的 bucket 开始遍历，并且
是从这个 bucket 的一个随机序号的 cell 开始遍历。这样，即使你是一个写死的 map，仅仅只是遍历它，也不太可能会返回一个固定序列的 key/value 对了。


# 7 map的key 可以是 float 型吗？
从语法上看，是可以的。Go 语言中只要是可比较的类型都可以作为 key。除开 slice，map，functions 这几种类型，其他类型都是 OK 的。具体包括：
布尔值、数字、字符串、指针、通道、接口类型、结构体、只包含上述类型的数组。这些类型的共同特征是支持 == 和 != 操作符，k1 == k2 时，
可认为 k1 和 k2 是同一个 key。如果是结构体，则需要它们的字段值都相等，才被认为是相同的 key。
顺便说一句，任何类型都可以作为 value，包括 map 类型。

来看个例子：
```golang
package main

import (
	"fmt"
	"math"
)

func main() {
   m := make(map[float64]int)
   m[1.4] = 1
   m[2.4] = 2
   m[math.NaN()] = 3
   m[math.NaN()] = 3

   for k, v := range m {
      fmt.Printf("[%v, %d] ", k, v)
   }

   fmt.Printf("\nk: %v, v: %d\n", math.NaN(), m[math.NaN()])
   fmt.Printf("k: %v, v: %d\n", 2.400000000001, m[2.400000000001])
   fmt.Printf("k: %v, v: %d\n", 2.4000000000000000000000001, m[2.4000000000000000000000001])

   fmt.Println(math.NaN() == math.NaN())
}
```
程序的输出:

```shell
[2.4, 2] [NaN, 3] [NaN, 3] [1.4, 1] 
k: NaN, v: 0
k: 2.400000000001, v: 0
k: 2.4, v: 2
false
```

例子中定义了一个 key 类型是 float 型的 map，并向其中插入了 4 个 key：1.4， 2.4， NAN，NAN。

打印的时候也打印出了 4 个 key，如果你知道 NAN != NAN，也就不奇怪了。因为他们比较的结果不相等，自然，在 map 看来
就是两个不同的 key 了。
接着，我们查询了几个 key，发现 NAN 不存在，2.400000000001 也不存在，而 2.4000000000000000000000001 却存在。
有点诡异，不是吗？
接着，我通过汇编发现了如下的事实：
当用 float64 作为 key 的时候，先要将其转成 uint64 类型，再插入 key 中。

具体是通过 `Float64frombits` 函数完成：

```golang
/ Float64frombits returns the floating point number corresponding
// the IEEE 754 binary representation b.
func Float64frombits(b uint64) float64 { return *(*float64)(unsafe.Pointer(&b)) }
```

我们再来输出点东西：

```golang
package main

import (
   "fmt"
   "math"
)

func main() {
   m := make(map[float64]int)
   m[2.4] = 2

   fmt.Println(math.Float64bits(2.4))
   fmt.Println(math.Float64bits(2.400000000001))
   fmt.Println(math.Float64bits(2.4000000000000000000000001))
}
```

输出：

```shell
4612586738352862003
4612586738352864255
4612586738352862003
```
可以看到，`2.4` 和 `2.4000000000000000000000001` 经过 `math.Float64bits()` 函数转换后的结果是一样的。自然，
二者在 map 看来，就是同一个 key 了。

所以我们的结论是：float 型可以作为 key，但是由于精度的问题，会导致一些诡异的问题，慎用之。

关于当 key 是引用类型时，判断两个 key 是否相等，需要 hash 后的值相等并且 key 的字面量相等。
由 @WuMingyu 补充的例子：

```golang
func TestT(t *testing.T) {
   type S struct {
      ID int
   }
   s1 := S{ID: 1}
   s2 := S{ID: 1}

   var h = map[*S]int {}
   h[&s1] = 1
   h[&s2] = 2
   t.Log(h[&s1])
   t.Log(h[&s2])
   t.Log(s1 == s2)
}
```

输出：

```shell
=== RUN   TestT
--- PASS: TestT (0.00s)
    endpoint_test.go:74: 1
    endpoint_test.go:75: 2
    endpoint_test.go:76: true
PASS

Process finished with exit code 0
```

# 8 可以边遍历边删除吗
map 并不是一个线程安全的数据结构。同时读写一个 map 是未定义的行为，如果被检测到，会直接 panic。
一般而言，这可以通过读写锁来解决：sync.RWMutex。
读之前调用 RLock() 函数，读完之后调用 RUnlock() 函数解锁；写之前调用 Lock() 函数，写完之后，调用 Unlock() 解锁。
另外，sync.Map 是线程安全的 map，也可以使用。

# 9 可以对 map 的元素取地址吗

无法对 map 的 key 或 value 进行取址。以下代码不能通过编译：

```golang

package main

import "fmt"

func main() {
   m := make(map[string]int)

   fmt.Println(&m["qcrao"])
}
```
编译报错:

```shell
./main.go:8:14: cannot take the address of m["qcrao"]
```
如果通过其他 hack 的方式，例如 unsafe.Pointer 等获取到了 key 或 value 的地址，也不能长期持有，因为一旦发生扩容，
key 和 value 的位置就会改变，之前保存的地址也就失效了。

# 10 如何判断两个map是否相等？

map 深度相等的条件：

- 都为 nil
- 非空、长度相等，指向同一个 map 实体对象
- 相应的 key 指向的 value “深度”相等

所谓深度相等，需要使用reflect.DeepEqual
直接将使用 map1 == map2 是错误的。这种写法只能比较 map 是否为 nil。

```golang

package main

import "fmt"

func main() {
   var m map[string]int
   var n map[string]int

   fmt.Println(m == nil)
   fmt.Println(n == nil)

   // 不能通过编译
   //fmt.Println(m == n)
}
```

```shell
true
true
```

因此只能是遍历map 的每个元素，比较元素是否都是深度相等。

总结一下，Go 语言中，通过哈希查找表实现 map，用链表法解决哈希冲突。
通过 key 的哈希值将 key 散落到不同的桶中，每个桶中有 8 个 cell。哈希值的低位决定桶序号，高位标识同一个桶中的不同 key。
当向桶中添加了很多 key，造成元素过多，或者溢出桶太多，就会触发扩容。扩容分为等量扩容和 2 倍容量扩容。扩容后，原来一个 bucket 中的
key 一分为二，会被重新分配到两个桶中。
扩容过程是渐进的，主要是防止一次扩容需要搬迁的 key 数量过多，引发性能问题。触发扩容的时机是增加了新元素，bucket 搬迁的时机则发生在赋值、
删除期间，每次最多搬迁两个 bucket。
查找、赋值、删除的一个很核心的内容是如何定位到 key 所在的位置，需要重点理解。

初始化map时预先设置好容量可以有效减少内存分配的次数，有利于提升性能。

```shell
BenchmarkWithoutPreAlloc-8      1000000000               0.0007040 ns/op               0 B/op          0 allocs/op
BenchmarkWithPreAlloc-8         1000000000               0.0003612 ns/op               0 B/op          0 allocs/op
PASS
```

不要使用float64或结构体作为map的key，因为前者由于精度的问题，会导致一些诡异的问题，后者如果是引用类型(结构体指针)，
判断两个 key 是否相等，需要 hash 后的值相等并且 key 的字面量相等。

map 不是线程安全的。在查找、赋值、遍历、删除的过程中都会检测写标志，一旦发现写标志置位（等于1），则直接 panic。赋值和删除函数在检测
完写标志是复位之后，先将写标志位置位，才会进行之后的操作。

检测写标志：

```golang
if h.flags&hashWriting == 0 {
      throw("concurrent map writes")
   }
```

设置写标志：

```golang
h.flags |= hashWriting
```

# 11 map gc优化

如果我们在本地缓存大量数据，如何避免 GC 导致的性能开销？

我们以下面的代码为例，分别测试 key、value 为 string 类型的 map 在不同数据规模下的 GC 开销。

```go
package optimize_gc

import (
	"fmt"
	"runtime"
	"testing"
	"time"
)

func GenerateStringMap(size int) map[string]string {
	// 在这里执行一些可能会触发GC的操作，例如创建大量对象等
	// 以下示例创建一个较大的map并填充数据
	m := make(map[string]string)
	for i := 0; i < size; i++ {
		key := fmt.Sprintf("key_%d", i)
		value := fmt.Sprintf("val_%d", i)
		m[key] = value

	}
	return m
}

// TestSmallBatchGCDuration 测试小规模数据gc时长
func TestSmallBatchGCDuration(t *testing.T) {
	size := 1000
	m := GenerateStringMap(size)
	runtime.GC()
	gcCost := timeGC()
	t.Logf("size %d GC duration: %v\n", size, gcCost)
	_ = m["1"]
}

// TestBigBatchGCDuration 测试大规模数据gc时长
func TestBigBatchGCDuration(t *testing.T) {
	size := 5000000
	m := GenerateStringMap(size)
	runtime.GC()
	gcCost := timeGC()
	t.Logf("size %d GC duration: %v\n", size, gcCost)
	_ = m["1"]
}

func timeGC() time.Duration {
	// 记录GC开始时间
	gcStartTime := time.Now()
	// 手动触发GC，以便更准确地测量此次操作相关的GC时长
	runtime.GC()
	// 计算总的GC时长
	gcCost := time.Since(gcStartTime)
	return gcCost
}

```

测试结果出来了，map 中储存 1k 条数据和 500w 条数据的 GC 耗时差异巨大。500w 条数据，GC 耗时 85ms，而 1k 条数据耗时
只需要 442µs，有近200 倍的性能差异。

```shell
 go test -gcflags=all=-l mp_gc_optimize_test.go -v
=== RUN   TestSmallBatchGCDuration
    mp_gc_optimize_test.go:29: size 1000 GC duration: 441.882µs
--- PASS: TestSmallBatchGCDuration (0.00s)
=== RUN   TestBigBatchGCDuration
    mp_gc_optimize_test.go:39: size 5000000 GC duration: 85.211619ms
--- PASS: TestBigBatchGCDuration (4.71s)
PASS
ok      command-line-arguments  5.459s
```

那么在大规模数据缓存下，GC 为什么耗时会这么长呢？这是因为 GC 在做对象扫描标记时，需要扫描标记 map 里面的全量 key-value 对象，
数据越多，需要扫描的对象越多，GC 时间也就越长。扫描标记的耗时过长，会引发一系列不良影响。它不仅会大量消耗 CPU 资源，
降低服务吞吐，而且在标记工作未能及时完成的情况下，GC 会要求处理请求的协程暂停手头的业务逻辑处理流程，转而协助 GC 
开展标记任务。这样一来，部分请求的响应延时将会不可避免地大幅升高，严重影响系统的响应效率与性能表现。为了避免 GC 
对程序性能造成影响，对于 map 类型，Golang 在 1.5 版本(https://go-review.googlesource.com/c/go/+/3288)
提供了一种绕过 GC 扫描的方法。绕过 GC 要满足下面两个条件。

> 第一，map 的 key-value 类型不能是指针类型且内部不能包含指针。比如 string 类型，它的底层数据结构中有指向数组的指针，
因此不满足这个条件。

```go
// 字符串数据结构
type stringStruct struct {
    str unsafe.Pointer //指针类型，指向字节数组
    len int
}
```

那到底不含指针类型，能不能缩短 GC 开销呢？咱们将代码里 map 的 key-value 类型换成 int 类型再试一下。

```go
func GenerateIntMap(size int) map[int]int {
	// 在这里执行一些可能会触发GC的操作，例如创建大量对象等
	// 以下示例创建一个较大的map并填充数据
	m := make(map[int]int)
	for i := 0; i < size; i++ {
		m[i] = i

	}
	return m
}

// 测试key-value非指针类型,int的gc开销
func TestBigBatchIntGCDuration(t *testing.T) {
	size := 5000000
	m := GenerateIntMap(size)
	runtime.GC()
	gcCost := timeGC()
	t.Logf("size %d GC duration: %v\n", size, gcCost)
	_ = m[1]
}
```

你会发现，key-value 换成 int 类型的 map，gc 性能提升非常明显，gc 时间从 85ms 变成了 不到2ms，提升42倍。

```shell
go test -gcflags=all=-l -run TestBigBatchIntGCDuration -v
=== RUN   TestBigBatchIntGCDuration
    mp_gc_optimize_test.go:60: size 5000000 GC duration: 1.926306ms
--- PASS: TestBigBatchIntGCDuration (1.24s)
PASS
ok      go-notes/goprincipleandpractise/map/optimizegc      1.810s
```

> 第二，key-value 除了需要满足非指针这个条件，key/value 的大小也不能超过 128 字节，如果超过 128 字节，
key-value 就会退化成指针，导致被 GC 扫描。

我们用 value 大小分别是 128、129 字节的结构体测试一下，测试代码如下。

```go
func TestSmallStruct(t *testing.T) {
	type SmallStruct struct {
		data [128]byte
	}
	m := make(map[int]SmallStruct)
	size := 5000000
	for i := 0; i < size; i++ {
		m[i] = SmallStruct{}
	}
	runtime.GC()
	gcCost := timeGC()
	t.Logf("size %d GC duration: %v\n", size, gcCost)
	_ = m[1]
}
func TestBigStruct(t *testing.T) {
	type BigStruct struct {
		data [129]byte
	}
	m := make(map[int]BigStruct)
	size := 5000000
	for i := 0; i < size; i++ {
		m[i] = BigStruct{}
	}
	runtime.GC()
	gcCost := timeGC()
	t.Logf("size %d GC duration: %v\n", size, gcCost)
	_ = m[1]
}
```

果然，key-value 的大小超过 128 字节会导致 GC 性能开销变大。对于 129 字节的结构体，GC 耗时 129ms，而 128 字节，
只需要 4.55ms，性能差距高达近30倍。

```shell
go test -gcflags=all=-l -run "TestSmallStruct|TestBigStruct" -v
=== RUN   TestSmallStruct
    mp_gc_optimize_test.go:75: size 5000000 GC duration: 4.552182ms
--- PASS: TestSmallStruct (2.57s)
=== RUN   TestBigStruct
    mp_gc_optimize_test.go:89: size 5000000 GC duration: 129.120292ms
--- PASS: TestBigStruct (2.17s)
PASS
ok      go-notes/goprincipleandpractise/map/optimizegc      5.522s
```

通过前面的测试，我们知道了，在缓存大规模数据时，为了避免 GC 开销，key-value 不能含指针类型且 key-value 的大小不能超过
128 字节。实际上，咱们在缓存大规模数据时，可以使用成熟的开源库来实现，比如 bigcache、freecache 等。它们的底层就是使用
分段锁加 map 类型来实现数据存储的，同时，它们也利用了刚刚讲过的 map 的 key-value 特性，来避免 GC 扫描。

以bigcache 为例，它的使用比较简单。通过 Get 和 Set 方法就可以实现读写操作。

```go
import (
     "fmt"
	 "context""github.com/allegro/bigcache/v3"
)

cache, _ := bigcache.New(context.Background(), bigcache.DefaultConfig(10 * time.Minute))
cache.Set("my-unique-key", []byte("value"))
entry, _ := cache.Get("my-unique-key")
fmt.Println(string(entry))
```

# 12 nil map 与 empty map 的行为差异

未初始化的 map（nil map）和已初始化的空 map（empty map）行为上有关键差异：

| 操作 | nil map | empty map |
|------|---------|-----------|
| 读 `m[key]` | 返回零值，不 panic | 返回零值 |
| `len(m)` | 0 | 0 |
| `range` | 不执行循环体 | 不执行循环体 |
| 写 `m[key] = v` | **panic** | 安全 |
| `delete(m, key)` | 安全（Go 1.0+） | 安全 |
| `m == nil` | true | false |

这是初学者最常见的 panic 来源之一。示例代码见 `trap/nil-map/main.go`：

```go
// nil map：仅声明未初始化
var m map[string]int
fmt.Println(m["key"])   // 0（零值，不 panic）

m["key"] = 1            // panic: assignment to entry in nil map

// empty map：通过 make 或字面量创建
m2 := make(map[string]int)
m2["key"] = 1           // 安全
```

最佳实践：始终通过 `make(map[K]V)` 或字面量 `map[K]V{}` 初始化 map，避免直接使用 `var m map[K]V` 后写入。

# 13 range 中 delete 是安全的

Go 语言规范明确保证：range 遍历 map 的过程中 delete 是安全的。

> "If a map entry that has not yet been reached is removed during iteration,
>  the corresponding iteration value will not be produced.
>  If a map entry is created during iteration, that entry may be produced
>  during the iteration or may be skipped."
>  — The Go Programming Language Specification

示例代码见 `trap/range-delete/main.go`：

```go
m := map[int]string{1: "a", 2: "b", 3: "c", 4: "d"}
for k := range m {
    if k%2 == 0 {
        delete(m, k) // 安全：删除当前或其他 key 均可
    }
}
fmt.Println(m) // map[1:a 3:c]
```

注意两点：
1. range 中 **delete 安全**，包括删除当前 key 和尚未遍历到的 key。
2. range 中 **insert 行为不确定**——新插入的 key 可能出现在后续迭代中，也可能不出现。因此应避免在 range 中向 map 插入新 key。

# 14 map 不能缩容

delete 操作只清除 key/value 并将 tophash 标记为 empty，**不会释放底层 bucket 数组的内存**。即使删除全部元素，已分配的 bucket 内存仍然保留。

实验代码见 `trap/no-shrink/no_shrink_one_test.go`，测试结果如下：

```shell
go test -run TestMapNoShrink -v
=== RUN   TestMapNoShrink
    no_shrink_one_test.go:23: 空 map 堆内存: 0.70 MB
    no_shrink_one_test.go:29: 填充 100w 后堆内存: 42.08 MB
    no_shrink_one_test.go:39: 删除全部后堆内存: 42.09 MB ← 内存未释放
    no_shrink_one_test.go:50: 新建 map 后堆内存: 0.67 MB ← 旧 map 被 GC 回收
--- PASS: TestMapNoShrink (0.12s)
```

填充 100 万条数据占用 42 MB，删除全部后仍占用 42 MB，直到创建新 map 替换旧 map 后才释放。

这意味着如果 map 曾经存储过大量数据后又大量删除，空桶占用的内存不会归还。生产环境中常见的场景：
- 缓存热点数据过期后大量删除
- 临时聚合任务结束后清空 map

解决方案：当 map 经历"大量写入后大量删除"的场景时，应创建新 map 并迁移存活数据，让旧 map 被 GC 回收：

```go
// 不要这样做
for k := range oldMap {
    delete(oldMap, k) // bucket 内存不会释放
}

// 应该这样做
newMap := make(map[K]V, len(survivingKeys))
for _, k := range survivingKeys {
    newMap[k] = oldMap[k]
}
oldMap = newMap // 旧 map 整体被 GC 回收
```

注意：测量 map 内存时需使用 `runtime.KeepAlive(m)` 防止 GC 在 `ReadMemStats` 之前提前回收 map 的内部存储。