
---
redis底层数据结构
---

简单来说，底层数据结构⼀共有6种，分别是简单动态字符串、双向链表、压缩列表、哈希表、跳表和整数
数组。它们和数据类型的对应关系如下图所⽰：





![redis-root-data-structure.png](images%2Fredis-root-data-structure.png)



# 1 SDS(简单动态字符串)


## 1.1 SDS结构设计
SDS 结构里包含了一个字节数组 buf[]，用来保存实际数据。同时，SDS 结构里还包含了三个元数据，分别是字节数组现有长度 len、
分配给字节数组的空间长度 alloc，以及 SDS 类型 flags。其中，Redis 给 len 和 alloc 这两个元数据定义了多种数据类型，进
而可以用来表示不同类型的 SDS，稍后我会给你具体介绍。下图显示了 SDS 的结构。
buf：字节数组，保存实际数据。为了表⽰字节数组的结束，Redis会⾃动在数组最后加⼀个“\0”，这就会额外占⽤1个字节的开销。
len：占4个字节，表⽰buf的已⽤⻓度。
alloc：也占个4字节，表⽰buf的实际分配⻓度，⼀般⼤于len





![sds.png](images%2Fsds.png)





## 1.2 SDS与C字符串的区别

### 1.2.1 常数级复杂度获取字符串长度
SDS 结构里的元数据有记录字符数组现有长度的len，所以SDS能以O(1)的时间复杂度获取SDS的长度，而C字符串则需要遍历整个字符串，
时间复杂度为O(N)。


### 1.2.2 杜绝缓冲区溢出

Redis 中实现字符串追加的函数是 sds.c 文件中的sdscatlen 函数，其执行过程如下：





![sdscatlen.png](images%2Fsdscatlen.png)





首先，获取目标字符串的当前长度，并调用 sdsMakeRoomFor 函数，根据当前长度和要追加的长度，判断是否要给目标字符串新增空间。
这一步主要是保证，目标字符串有足够的空间接收追加的字符串。 其次，在保证了目标字符串的空间足够后，将源字符串中指定长度 len 
的数据追加到目标字符串。 最后，设置目标字符串的最新长度。

也就是说，当redis中要对字符串进行追加(修改)时，它会首先检查SDS的已分配空间alloc是否满足追加的需求，如果不满足会自动将SDS的
空间扩展至执行追加所需的大小，然后才会执行实际的追加操作，这样就杜绝了因为空间不足导致的缓冲区溢出的问题。

而C字符串没有记录自身的长度，所以其strcat函数会假定用户调用该函数时，已经为目标字符串分配了足够的内存，足以容纳源字符串以及追加
的内容，然而一旦用户没有根据当前长度和要追加的长度，给目标字符串新增空间，就会产生缓冲区溢出。


### 1.2.3 减少修改字符串时带来的内存重分配次数

为了避免频繁进行耗时的内存重分配操作，影响redis性能，redis使用未使用空间来实现空间预分配和惰性空间释放。如果SDS的已分配空间alloc
已满足追加的需求，那么就直接使用未使用空间存储追加数据，不需要再执行内存重分配，如果不满足，程序不仅会为SDS分配追加操作所必需的
空间，还会为SDS分配额外的未使用空间。通过这种预分配策略，SDS将连续增长N次字符串所需的内存重分配次数从必定N次降低为最多N次。

惰性空间释放用于优化SDS字符串的缩短操作，当需要缩短时，程序并不会立即使用内存重分配来回收缩短后空出来的字节，而是将其作为未使用
空间保留在SDS里面，如果将来要对SDS进行追加操作，这些未使用空间就可能派上用场。

**扩容策略**
SDS字符串在⻓度⼩于 1M 之前， 扩容空间采⽤加倍策略， 也就是保留 100% 的冗余空间。 当⻓度超过 1M 之后， 为了避免加倍后的冗余空
间过⼤⽽导致浪费， 每次扩容只会多分配 1M ⼤⼩的冗余空间。


### 1.2.4 二进制安全

我们知道，C字符串中的字符必须符合某种编码(比如ASCII)，并且除了字符串的末尾之外，字符串里不能包含空字符，否则最先被程序读入的空字符
会被误认为是字符串结尾，这些限制使得C字符串只能保存文本数据，而不能保存像图片、音频、视频、压缩文件这样的二进制数据。
而SDS的API都是二进制安全的，所有SDS的API都会以二进制的方式来处理SDS存放在buf字节数组里的数据，程序不会对其中的数据做任何限制、过滤、
或者假设，数据在写入时是什么样的，被读取时就是什么样的。因此，用SDS来保存包含多个空字符的数据就没问题，因为SDS使用len属性的值而不是
空字符来判断字符串是否结束。所以，redis的SDS不仅可以保存文本数据，还可以保存任意格式的二进制数据。

### 1.2.5 兼容部分C字符串函数

虽然SDS的API都是二进制安全的，但它们一样遵循C字符串以空字符串结尾的惯例。这些API总会将SDS保存的数据的末尾设置为空字符，并且总会在
为buf字节数组分配空间时多分配一个字节来容纳这个空字符，这是为了让那些保存文本数据的SDS可以重用一部分C字符串的库函数。


## 1.3 字符串编码

字符串对象的编码可以是int, raw或者embstr。
当保存64位有符号整数时，String类型会把它保存为⼀个8字节的Long类型整数，这种保存⽅式通常也叫作int编码⽅式。
但是，当保存的数据中包含字符时，String类型就会⽤简单动态字符串（Simple Dynamic String，SDS）结构体来保存。


可以看到，在SDS中，buf保存实际数据，⽽len和alloc本⾝其实是SDS结构体的额外开销。 另外，对于String类型来说，除了SDS的额外开销，
还有⼀个来⾃于RedisObject结构体的开销。 因为Redis的数据类型有很多，⽽且，不同数据类型都有些相同的元数据要记录（⽐如最后⼀次访问的时
间lru、被引⽤的次数refcount等），所以，Redis会⽤⼀个RedisObject结构体来统⼀记录这些元数据，同时指向实际数据。
⼀个RedisObject包含了8字节的元数据和⼀个8字节指针，这个指针再进⼀步指向具体数据类型的实际数据所在，例如指向String类型的SDS结构所在的
内存地址，可以看⼀下下⾯的⽰意图。





![redis_object.png](images%2Fredis_object.png)





为了节省内存空间，Redis还对Long类型整数和SDS的内存布局做了专⻔的设计。 ⼀⽅⾯，当保存的是Long类型整数时，RedisObject中的指针就直接赋值
为整数数据了，这样就不⽤额外的指针再指向整数了，节省了指针的空间开销。

另⼀⽅⾯，当保存的是字符串数据，并且字符串⼩于等于44字节时，RedisObject中的元数据、指针和SDS是⼀块连续的内存区域，这样就可以避免内存碎⽚。
这种布局⽅式也被称为embstr编码⽅式。

当然，当字符串⼤于44字节时，SDS的数据量就开始变多了，Redis就不再把SDS和RedisObject布局在⼀起了，⽽是会给SDS分配独⽴的空间，并⽤指针指向
SDS结构。这种布局⽅式被称为raw编码模式。

为了帮助你理解int、embstr和raw这三种编码模式，我画了⼀张⽰意图，如下所⽰：





![string-encoding.png](images%2Fstring-encoding.png)




# 2 从ziplist到quicklist，再到listpack

## 2.1 ziplist(压缩列表)的设计

Redis 为了节约内存空间使⽤， zset 和 hash 容器对象在元素个数较少的时候， 采⽤压缩列表 (ziplist) 进⾏存储。 压缩列表是⼀块连
续的内存空间，元素之间紧挨着存储，没有任何冗余空隙。

```C
struct ziplist<T> {
int32 zlbytes; // 整个压缩列表占⽤字节数
int32 zltail_offset; // 最后⼀个元素距离压缩列表起
始位置的偏移量， ⽤于快速定位到最后⼀个节点
int16 zllength; // 元素个数
T[] entries; // 元素内容列表， 挨个挨个紧凑存储
int8 zlend; // 标志压缩列表的结束， 值恒为 0xFF
}
```

```C
struct entry {
int<var> prevlen; // 前⼀个 entry 的字节⻓度
int<var> encoding; // 元素类型编码
optional byte[] content; // 元素内容
}
```




![ziplist.png](images%2Fziplist.png)





压缩列表为了⽀持双向遍历， 所以才会有 ztail_offset 这个字段， ⽤来快速定位到最后⼀个元素，然后根据prevlen，也就是前一个entry的长度，
倒着遍历。

entry的 prevlen 字段表示前⼀个 entry 的字节⻓度， 当压缩列表倒着遍历时， 需要通过这个字段来快速定位到下⼀个元素的位置。 它是⼀
个变⻓的整数， 当字符串⻓度⼩于 254(0xFE) 时， 使⽤⼀个字节表示； 如果达到或超出 254(0xFE) 那就使⽤ 5 个字节来表示。 第⼀个
字节是 0xFE(254)， 剩余四个字节表示字符串⻓度。 

encoding字段存储了元素内容的编码类型信息， ziplist 通过这个字段来决定后⾯的 content 内容的形式。


**增加元素**

因为 ziplist 都是紧凑存储， 没有冗余空间 (对⽐⼀下 Redis 的字符串结构)。 意味着插⼊⼀个新的元素就需要调⽤ realloc 扩展内存。
取决于内存分配器算法和当前的 ziplist 内存⼤⼩， realloc 可能会重新分配新的内存空间， 并将之前的内容⼀次性拷⻉到新的地址， 也
可能在原有的地址上进⾏扩展， 这时就不需要进⾏旧内容的内存拷⻉。

如果 ziplist 占据内存太⼤， 重新分配内存和拷⻉内存就会有很⼤的消耗。 所以 ziplist 不适合存储⼤型字符串， 存储的元素也不宜过多。


**级联更新**

前⾯提到每个 entry 都会有⼀个 prevlen 字段存储前⼀个 entry 的⻓度。 如果内容⼩于 254 字节， prevlen ⽤ 1 字节存储， 否则就
是 5 字节。 这意味着如果某个 entry 经过了修改操作从 253 字节变成了 254 字节，那么它的下⼀个 entry 的 prevlen 字段就要更
新， 从 1 个字节扩展到 5 个字节； 如果这个 entry 的⻓度本来也是253 字节， 那么后⾯ entry 的 prevlen 字段还得继续更新。
如果 ziplist ⾥⾯每个 entry 恰好都存储了 253 字节的内容， 那么第⼀个 entry 内容的修改就会导致后续所有 entry 的级联更新， 这
就是⼀个⽐较耗费计算资源的操作。 因为级联更新在最坏情况下需要对压缩列表执行N次空间重分配操作，而每次空间重分配操作的最坏时间复杂度
为O(N)，所以级联更新的最坏复杂度为O(N*N) 同理，删除中间的某个节点也可能会导致级联更新。

但是这两种操作出现的几率并不高，首先压缩列表里要恰好有多个连续的，长度介于250~253字节之间的节点，级联更新才可能被引发，在实际中，
这种情况并不多见。
其次，即使出现级联更新，但只要被更新的节点数量不多，就不会对性能造成太大影响。


**查找复杂度高**

因为 ziplist 头尾元数据的大小是固定的，并且在 ziplist 头部记录了最后一个元素的位置， 所以，当在 ziplist 中查找第一个或最后一个元素的时候，
就可以很快找到。 但问题是，当要查找列表中间的元素时，ziplist 就得从列表头或列表尾遍历才行。而当ziplist 保存的元素过多时，查找中间数据的
复杂度就增加了。更糟糕的是，如果 ziplist 里面保存的是字符串，ziplist 在查找某个元素时，还需要逐一判断元素的每个字符，这样又进一步增加了
复杂度。 也正因为如此，我们在使用 ziplist 保存 Hash 或 Sorted Set 数据时，都会在 redis.conf文件中，通过 hash-max-ziplist-entries 
和 zset-max-ziplist-entries 两个参数，来控制保存在 ziplist 中的元素个数。



## 2.2 quicklist的设计与实现

quicklist 的设计，其实是结合了链表和 ziplist 各自的优势。简单来说，一个 quicklist 就是一个链表，而链表中的每个元素又是一个 ziplist。

首先，quicklist 元素的定义，也就是 quicklistNode。因为 quicklist 是一个链表，所以每个 quicklistNode 中，都包含了分别指向它前序和
后序节点的指针*prev和*next。同时，每个 quicklistNode 又是一个 ziplist，所以，在 quicklistNode 的结构体中，还有指向 ziplist 的指针*zl。
此外，quicklistNode 结构体中还定义了一些属性，比如 ziplist 的字节大小、包含的元素个数、编码格式、存储方式等。下面的代码显示了 quicklistNode
的结构体定义。

```C
typedef struct quicklistNode {
struct quicklistNode *prev; //前一个quicklistNode
struct quicklistNode *next; //后一个quicklistNode
unsigned char *zl; //quicklistNode指向的ziplist
unsigned int sz; //ziplist的字节大小
unsigned int count : 16; //ziplist中的元素个数
unsigned int encoding : 2; //编码格式，原生字节数组或压缩存储
unsigned int container : 2; //存储方式
unsigned int recompress : 1; //数据是否被压缩
unsigned int attempted_compress : 1; //数据能否被压缩
unsigned int extra : 10; //预留的bit位
} quicklistNode;
```

了解了 quicklistNode 的定义，我们再来看下 quicklist 的结构体定义。

quicklist 作为一个链表结构，在它的数据结构中，是定义了整个 quicklist 的头、尾指针，这样一来，我们就可以通过 quicklist 的数据结构，
来快速定位到 quicklist 的链表头和链表尾。
此外，quicklist 中还定义了 quicklistNode 的个数、所有 ziplist 的总元素个数等属性。quicklist 的结构定义如下所示：

```C
typedef struct quicklist {
quicklistNode *head; //quicklist的链表头
quicklistNode *tail; //quicklist的链表尾
unsigned long count; //所有ziplist中的总元素个数
unsigned long len; //quicklistNodes的个数
...
} quicklist
```

然后，从 quicklistNode 和 quicklist 的结构体定义中，我们就能画出下面这张 quicklist 的示意图。





![quicklist.png](images%2Fquicklist.png)





而也正因为 quicklist 采用了链表结构，所以当插入一个新的元素时，quicklist 首先就会检查插入位置的 ziplist 是否能容纳该元素，
这是通过 _quicklistNodeAllowInsert 函数来完成判断的。
_quicklistNodeAllowInsert 函数会计算新插入元素后的大小（new_sz），这个大小等于 quicklistNode 的当前大小（node->sz）、
插入元素的大小（sz），以及插入元素后ziplist 的 prevlen 占用大小。

在计算完大小之后，_quicklistNodeAllowInsert 函数会依次判断新插入的数据大小（sz） 是否满足要求，即单个 ziplist 是否不超过 8KB，
或是单个 ziplist 里的元素个数是否满足要求。

只要这里面的一个条件能满足，quicklist 就可以在当前的 quicklistNode 中插入新元素， 否则 quicklist 就会新建一个 quicklistNode，
以此来保存新插入的元素。

这样一来，quicklist 通过控制每个 quicklistNode 中，ziplist 的大小或是元素个数，就有效减少了在 ziplist 中新增或修改元素后，
发生连锁更新的情况，从而提供了更好的访问性能。

因为 quicklist 使用 quicklistNode 结构指向每个 ziplist，无疑增加了内存开销。 为了减少内存开销，并进一步避免 ziplist 连锁更新问题，
Redis 在 5.0 版本中，就设计实现了 listpack 结构。


## 2.3 listpack设计与实现

listpack 也叫紧凑列表，它的特点就是用一块连续的内存空间来紧凑地保存数据，同时为了节省内存空间，listpack 列表项使用了多种编码方式，
来表示不同长度的数据，这些数据包括整数和字符串。

下面这张图展示了listpack的整体结构。






![listpack.png](images%2Flistpack.png)


好了，了解了 listpack 的整体结构后，我们再来看下 listpack 列表项的设计。

和 ziplist 列表项类似，listpack 列表项也包含了元数据信息和数据本身。不过，为了避免ziplist 引起的连锁更新问题，listpack 中的每个
列表项不再像 ziplist 列表项那样，保存其前一个列表项的长度，它只会包含三个方面内容，分别是当前元素的编码类型（entryencoding）、
元素数据 (entry-data)，以及编码类型和元素数据这两部分的长度 (entrylen)，如下图所示。





![listpack-entry.png](images%2Flistpack-entry.png)



**listpack 避免级联更新的实现方式**


在 listpack 中，因为每个列表项只记录自己的长度，而不会像 ziplist 中的列表项那样，会记录前一项的长度。所以，当我们在 listpack 中
新增或修改元素时，实际上只会涉及每个列表项自己的操作，而不会影响后续列表项的长度变化，这就避免了级联更新。




从左向右遍历 listpack 的基本过程
![listpack-search-from-left.png](images%2Flistpack-search-from-left.png)


如果是从右向左反向查询 listpack
首先，我们根据 listpack 头中记录的 listpack 总长度，就可以直接定位到 listapck 的尾部结束标记。然后，我们可以调用 lpPrev 函数，
该函数的参数包括指向某个列表项的指针， 并返回指向当前列表项前一项的指针。 lpPrev 函数中的关键一步就是调用 lpDecodeBacklen 函数。
lpDecodeBacklen 函数会从右向左，逐个字节地读取当前列表项的 entry-len。

那么，lpDecodeBacklen 函数如何判断 entry-len 是否结束了呢？

这就依赖于 entry-len 的编码方式了。entry-len 每个字节的最高位，是用来表示当前字节是否为 entry-len 的最后一个字节，这里存在两种
情况，分别是：
最高位为 1，表示 entry-len 还没有结束，当前字节的左边字节仍然表示 entry-len 的内容；
最高位为 0，表示当前字节已经是 entry-len 最后一个字节了

而 entry-len 每个字节的低 7 位，则记录了实际的长度信息。这里你需要注意的是，entry-len 每个字节的低 7 位采用了大端模式存储，
也就是说，entry-len 的低位字节保存在内存高地址上。

下面这张图，展示了 entry-len 这种特别的编码方式：





![entry-len-encoding.png](images%2Fentry-len-encoding.png)





实际上，正是因为有了 entry-len 的特别编码方式，lpDecodeBacklen 函数就可以从当前列表项起始位置的指针开始，向左逐个字节解析，
得到前一项的 entry-len 值。这也是lpDecodeBacklen 函数的返回值。而从刚才的介绍中，我们知道 entry-len 记录了编码类型和
实际数据的长度之和。
因此，lpPrev 函数会再调用 lpEncodeBacklen 函数，来计算得到 entry-len 本身长度， 这样一来，我们就可以得到前一项的总长度，
而 lpPrev 函数也就可以将指针指向前一项的起始位置了。所以按照这个方法，listpack 就实现了从右向左的查询功能。


# 3 双向链表

除了列表(list)键外，发布与订阅、慢查询、监视器等功能也用到了链表，redis服务器本身还使用链表来保存多个客户端的状态信息，以及使用
链表来构建客户端输出缓冲区。

链表的结构定义如下

```C
typedef struct list {
listNode *head; // 链表表头节点
listNode *tail; // 链表表尾节点
unsigned long len; // 链表所包含的节点数量
void *(*dup) (void *ptr);
void (*free) (void *ptr);
int (*match) (void *ptr, void *key);
} list
```

链表中的节点结构定义如下：

```C
typedef struct listNode {
struct listNode *prev; // 前驱结点
struct listNode *next; // 后继节点
void *value;           // 节点的值
} listNode;
```

list结构为链表提供了表头指针head、表尾指针tail，以及链表长度计数器len，而dup、free和match成员则是用于实现多态链表所需的类型
特定函数：
dup函数用于复制链表节点所保存的值；
free函数用于释放链表节点所保存的值；
match函数则用于对比链表节点所保存的值和另一个输入值是否相等。

redis的链表实现的特性可以总结如下:

双向: 链表节点带有prev和next指针，获取某个节点的前驱结点和后继节点的时间复杂度都是O(1)。
无环: 表头节点的prev指针和表尾节点的next指针都指向NULL，对链表的访问以NULL为终点。
带表头和表尾指针: 通过list结构的head和tail指针，获取链表头节点和尾节点的时间复杂度都是O(1)。
带链表长度计数器: 程序使用list结构的len属性对其持有的链表节点进行计数，获取链表中节点数量的时间复杂度都是O(1)。
多态: 链表节点使用void*指针来保存节点值，并且可以通过list结构list结构dup、free、match三个属性为节点值设置类型特定函数，
所以链表可以用于保存各种不同类型的值。


# 4 哈希表

Redis 的哈希表是整个数据库的核心数据结构。Redis 的全局键空间就是一个哈希表，所有的键值对都存储在其中。此外，Hash 类型在元素较多时，
底层编码也会使用哈希表。


## 4.1 哈希表结构设计

Redis 的哈希表由 `dict.h/dictht` 结构定义：

```C
typedef struct dictht {
    dictEntry **table;      // 哈希桶数组，每个元素是指向 dictEntry 的指针
    unsigned long size;      // 哈希表大小（桶数组长度），总是 2 的幂
    unsigned long sizemask;  // 大小掩码，等于 size - 1，用于计算索引
    unsigned long used;      // 已有节点数量
} dictht;
```

哈希桶数组中的每个元素是一个指向 `dictEntry` 的指针，`dictEntry` 的结构如下：

```C
typedef struct dictEntry {
    void *key;              // 键
    union {
        void *val;          // 值（指针）
        uint64_t u64;       // 值（无符号整数）
        int64_t s64;        // 值（有符号整数）
        double d;           // 值（浮点数）
    } v;
    struct dictEntry *next; // 指向下一个哈希节点，形成链表（解决冲突）
} dictEntry;
```

可以看到，`dictEntry` 通过 `next` 指针将多个哈希值相同的键值对串联成一个链表，这就是 Redis 解决哈希冲突的方式——**链式寻址法**。
当有新的键值对发生冲突时，Redis 会将新节点**插入到链表的头部**（头插法），因为头插法的时间复杂度为 O(1)。


## 4.2 哈希冲突与负载因子

**哈希冲突**是指两个或多个键被哈希函数映射到同一个桶位置。Redis 使用 MurmurHash2 算法计算键的哈希值，该算法即使输入的键是有规律的，
也能给出一个很好的随机分布性，并且计算速度也非常快。

随着操作的不断执行，哈希表中保存的键值对会逐渐增多或减少，为了让哈希表的**负载因子**（load factor）维持在一个合理的范围内，程序需要
对哈希表的大小进行相应的扩展或收缩。负载因子的计算公式为：

```
负载因子 = 哈希表已保存节点数量 / 哈希表大小
         = ht[0].used / ht[0].size
```

**扩容条件**（满足任一即触发）：
- 服务器目前**没有**执行 BGSAVE 或 BGREWRITEAOF 命令，并且负载因子 >= 1
- 服务器目前**正在**执行 BGSAVE 或 BGREWRITEAOF 命令，并且负载因子 >= 5

之所以在执行持久化操作时提高扩容阈值，是因为 BGSAVE 和 BGREWRITEAOF 都需要 fork 子进程，而操作系统使用写时复制（Copy-On-Write）
技术来优化子进程的使用效率。在子进程存在期间，尽量避免对哈希表执行扩容操作，可以减少不必要的内存写入，最大限度地节约内存。

**缩容条件**：当负载因子 < 0.1 时，程序自动对哈希表执行收缩操作。


## 4.3 渐进式 rehash

为了支持 rehash 操作，Redis 在更上层用 `dict` 结构体来封装两个哈希表：

```C
typedef struct dict {
    dictType *type;     // 类型特定函数
    void *privdata;     // 私有数据
    dictht ht[2];       // 两个哈希表，ht[1] 仅在 rehash 时使用
    long rehashidx;     // rehash 进度索引，-1 表示未进行 rehash
    int16_t pauserehash;// rehash 是否暂停（>0 表示暂停）
} dict;
```

普通状态下，所有数据都存储在 `ht[0]` 中，`ht[1]` 为空。当需要扩容或缩容时，Redis 并不会一次性将 `ht[0]` 中的所有键值对全部 rehash 到
`ht[1]`，而是采用**渐进式 rehash**，将 rehash 操作分散到对字典的每次增删改查操作中，避免集中式 rehash 带来的性能抖动。

**渐进式 rehash 的详细步骤**：

1. 为 `ht[1]` 分配空间：如果是扩容，`ht[1]` 的大小为第一个 >= `ht[0].used * 2` 的 2^n；如果是缩容，`ht[1]` 的大小为第一个 >= `ht[0].used` 的 2^n
2. 将 `rehashidx` 设置为 0，表示 rehash 开始
3. 在 rehash 期间，每次对字典执行增删改查操作时，程序除了执行指定的操作外，还会顺带将 `ht[0]` 在 `rehashidx` 索引位置上的所有键值对 rehash 到 `ht[1]`，完成后将 `rehashidx` 加 1
4. 随着字典操作的不断执行，最终 `ht[0]` 的所有键值对都会被 rehash 到 `ht[1]`
5. 将 `rehashidx` 设置为 -1，释放 `ht[0]`，将 `ht[1]` 设置为 `ht[0]`，并在 `ht[1]` 新建一个空白哈希表，为下一次 rehash 做准备

**rehash 期间的操作规则**：
- **查找**：先在 `ht[0]` 中查找，如果没找到再去 `ht[1]` 中查找
- **新增**：一律插入到 `ht[1]` 中，保证 `ht[0]` 只减不增，最终变为空表
- **删除和更新**：在两个表中都可能执行


## 4.4 哈希表在 Redis 中的应用

哈希表在 Redis 中有两个主要用途：

**1. 全局键空间**：Redis 使用一个全局的 `dict` 结构来保存所有键值对。每当执行 SET、GET、DEL 等命令时，实际上都是在操作这个全局哈希表。
这也是 Redis 能够以 O(1) 时间复杂度完成大多数单键操作的根本原因。

**2. Hash 类型的底层编码**：当 Hash 类型的元素个数超过 `hash-max-ziplist-entries`（默认512）或者某个元素的值长度超过 `hash-max-ziplist-value`
（默认64字节）时，Hash 类型的底层编码会从 ziplist（或 listpack）转换为 hashtable。


# 5 跳表

跳表（skiplist）是一种有序数据结构，它通过在每个节点中维持多个指向其他节点的指针，从而达到快速访问节点的目的。跳表支持平均 O(logN)、
最坏 O(N) 复杂度的查找，大部分情况下效率可以和平衡树相媲美，而且实现比平衡树更加简单。

Redis 使用跳表作为 Sorted Set（有序集合）的底层实现之一。当有序集合包含的元素数量较多，或者元素的成员是比较长的字符串时，Redis 就会
使用跳表来作为有序集合的底层实现。


## 5.1 跳表结构设计

跳表的核心思想是在有序链表的基础上，**建立多层索引**，通过逐层跳跃来加速查找过程。普通有序链表的查找时间复杂度为 O(N)，而通过增加多级
索引，跳表将查找时间复杂度降低到了 O(logN)。

Redis 中跳表节点的结构定义如下：

```C
typedef struct zskiplistNode {
    sds ele;                          // 成员对象（SDS 字符串）
    double score;                     // 分值
    struct zskiplistNode *backward;   // 后退指针，指向前一个节点
    struct zskiplistLevel {
        struct zskiplistNode *forward; // 前进指针，指向同层的下一个节点
        unsigned long span;            // 跨度，记录前进指针跨越了多少个节点
    } level[];                         // 层数组，柔性数组
} zskiplistNode;
```

各字段说明：
- **ele**：节点保存的成员对象，是一个 SDS 字符串
- **score**：节点保存的分值，跳表中的节点按分值从小到大排列
- **backward**：后退指针，用于从表尾向表头方向遍历，每个节点只有一个后退指针，所以每次只能后退一个节点
- **level[]**：层数组，每个层包含一个前进指针和一个跨度。层的数量在创建节点时随机生成（1~32之间）

跳表本身的结构定义如下：

```C
typedef struct zskiplist {
    struct zskiplistNode *header, *tail; // 头节点和尾节点
    unsigned long length;                // 节点数量（不包含头节点）
    int level;                           // 当前跳表中层数最大的节点的层数
} zskiplist;
```

头节点是一个特殊节点，它的 level 数组固定为 32 层（ZSKIPLIST_MAXLEVEL），不保存任何成员数据，仅作为各层链表的起始入口。


## 5.2 跳表的查找过程

跳表的查找从**最高层**开始，逐层向下查找，具体步骤如下：

1. 从头节点的最高层开始
2. 在当前层，沿着前进指针向右移动，直到遇到比目标值大的节点（或到达 NULL）
3. 退回到前一个节点，下降一层
4. 重复步骤 2-3，直到在第 1 层找到目标节点或确认目标不存在

这个过程类似于在多级索引中做二分搜索。由于每一层的节点数量大约是下一层的一半（概率意义上），所以查找的期望时间复杂度为 **O(logN)**。

**跨度（span）的作用**：跨度记录了两个节点之间的距离，它的主要用途是计算元素的排名（rank）。在查找过程中，将沿途访问过的所有层的跨度
累加起来，就可以得到目标节点在跳表中的排位。这使得 ZRANK 命令可以在 O(logN) 时间内完成。


## 5.3 节点层数的随机化

跳表的每个节点在创建时，通过一个**随机算法**来决定层的数量。Redis 使用一个幂次定律（power law）分布来生成层数：

```C
int zslRandomLevel(void) {
    int level = 1;
    while ((random() & 0xFFFF) < (ZSKIPLIST_P * 0xFFFF))
        level += 1;
    return (level < ZSKIPLIST_MAXLEVEL) ? level : ZSKIPLIST_MAXLEVEL;
}
```

其中 `ZSKIPLIST_P` 的值为 0.25，`ZSKIPLIST_MAXLEVEL` 的值为 32。这意味着：
- 每个节点至少有 1 层
- 节点有第 2 层的概率为 25%
- 节点有第 3 层的概率为 6.25%（25% × 25%）
- 节点有第 k 层的概率为 (1/4)^(k-1)
- 层数上限为 32 层

选择 p=0.25 而不是 p=0.5 是 Redis 做出的一个权衡：p=0.25 使得跳表每个节点平均只有 1.33 个指针（而 p=0.5 是 2 个），这意味着
更少的内存消耗，虽然查找时比较次数稍多，但 Redis 的作者认为这是一个更好的平衡点。


## 5.4 为什么用跳表而不用平衡树

Redis 的作者 antirez 曾解释过选择跳表的原因，主要有以下几点：

**1. 实现简单**：跳表的实现和调试都比平衡树（如红黑树、AVL树）简单得多。平衡树的插入和删除需要通过旋转来维护平衡性，代码复杂且容易
出错。而跳表只需要随机生成层数，再更新相应的指针即可。

**2. 范围查询高效**：有序集合经常需要执行 ZRANGEBYSCORE 等范围查询操作。跳表在找到起始节点后，只需要沿着第 1 层的前进指针依次遍历
就能获取一个范围内的所有元素，操作非常直观。而平衡树做范围查找则需要中序遍历，实现起来更复杂。

**3. 内存可控**：通过调整 p 值，可以灵活地在查找效率和内存消耗之间做权衡。Redis 选择 p=0.25，使得平均每个节点只有约 1.33 个前进指针，
内存开销很低。

**4. 插入和删除简单**：跳表的插入和删除只需要修改相邻节点的指针，不需要像平衡树那样做全局的平衡调整。


## 5.5 跳表在 Redis 中的应用

跳表在 Redis 中仅用于 Sorted Set（有序集合）的底层实现。当有序集合同时满足以下两个条件时，使用 ziplist（或 listpack）编码：
- 有序集合保存的元素数量小于 `zset-max-ziplist-entries`（默认128）
- 有序集合保存的所有元素的成员长度都小于 `zset-max-ziplist-value`（默认64字节）

不满足以上任一条件时，有序集合就会使用跳表作为底层实现。实际上，此时 Redis 会同时使用一个 `dict` 和一个 `zskiplist` 来实现有序集合：
- **dict**：用于以 O(1) 复杂度根据成员查找对应的分值（ZSCORE 命令）
- **zskiplist**：用于以 O(logN) 复杂度支持范围查询和排名操作（ZRANGEBYSCORE、ZRANK 等命令）

两种数据结构通过指针共享元素的成员和分值，不会产生额外的内存浪费。


# 6 整数集合

整数集合（intset）是 Redis 中 Set（集合）类型的底层实现之一。当一个集合只包含整数值元素，并且元素数量不多时，Redis 就会使用整数集合
作为集合的底层实现。整数集合的优势在于它以紧凑的方式存储整数，比哈希表节省大量内存。


## 6.1 整数集合结构设计

整数集合的结构定义如下：

```C
typedef struct intset {
    uint32_t encoding;  // 编码方式（决定 contents 数组中每个元素的类型）
    uint32_t length;    // 集合包含的元素数量
    int8_t contents[];  // 保存元素的数组（实际类型由 encoding 决定）
} intset;
```

虽然 `contents` 属性声明为 `int8_t` 类型的数组，但 `contents` 数组中每个元素的真正类型取决于 `encoding` 属性的值：

| encoding 值 | contents 中每个元素的类型 | 每个元素占用字节数 |
|---|---|---|
| INTSET_ENC_INT16 | int16_t | 2 字节 |
| INTSET_ENC_INT32 | int32_t | 4 字节 |
| INTSET_ENC_INT64 | int64_t | 8 字节 |

**整数集合的特性**：
- `contents` 数组中的元素按照**从小到大的顺序**排列
- 数组中**不包含任何重复元素**
- 紧凑的内存布局，没有任何冗余空间


## 6.2 编码升级机制

当我们将一个新元素添加到整数集合中，并且新元素的类型比整数集合现有所有元素的类型都要长时，整数集合需要先进行**升级**（upgrade），
然后才能将新元素添加进来。

**升级的详细步骤**：

1. 根据新元素的类型，扩展整数集合底层数组的空间大小，并为新元素分配空间
2. 将底层数组中现有的所有元素都转换成与新元素相同的类型，并将转换后的元素放置到正确的位置上，同时保持底层数组的有序性不变
3. 将新元素添加到底层数组中

例如，假设一个整数集合当前的编码为 INTSET_ENC_INT16，包含三个 int16_t 类型的元素 [1, 2, 3]，共占用 3 × 2 = 6 字节。
现在要插入一个 int32_t 类型的值 65535：

1. 将数组空间从 6 字节扩展到 4 × 4 = 16 字节（4个 int32_t 元素）
2. 从后向前，将原有元素 3、2、1 依次转换为 int32_t 类型并放到新的正确位置
3. 将 65535 插入到有序位置
4. 将 `encoding` 更新为 INTSET_ENC_INT32，`length` 更新为 4

**升级的好处**：

- **灵活性**：C 语言是静态类型语言，通常不会将两种不同类型的值放在同一个数据结构中。整数集合通过自动升级底层数组来适应新元素，
  不必担心类型不匹配的问题
- **节约内存**：如果所有元素都可以用 int16_t 表示，就不必使用 int64_t 来存储，只在需要时才升级，最大限度地节约了内存

**不支持降级**：一旦对数组进行了升级，编码就会一直保持升级后的状态。即使后来移除了所有需要大类型的元素，整数集合也不会执行降级操作。
这是因为降级需要遍历所有元素来确定是否可以降级，开销较大，而且 Redis 认为升级后再降级的场景较为少见。


## 6.3 查找与插入

**查找**：由于整数集合中的元素是有序排列的，所以查找操作使用**二分查找**算法，时间复杂度为 O(logN)。

**插入**：插入一个新元素的步骤为：
1. 检查新元素是否需要升级编码类型，如果需要，先执行升级
2. 使用二分查找定位新元素应该插入的位置，时间复杂度 O(logN)
3. 将插入位置之后的所有元素向后移动一位，为新元素腾出空间，时间复杂度 O(N)
4. 将新元素放入空出的位置
5. 更新 `length` 属性

因此，插入操作的总体时间复杂度为 **O(N)**（由数组元素移动主导）。

**删除**：与插入类似，删除某个元素后，需要将该位置之后的所有元素向前移动一位，时间复杂度同样为 O(N)。


## 6.4 整数集合在 Redis 中的应用

整数集合是 Set 类型的底层编码之一。当 Set 同时满足以下两个条件时，使用 intset 编码：
- 集合中所有元素都是整数值
- 集合中的元素数量不超过 `set-max-intset-entries`（默认512）

不满足以上任一条件时，Set 的底层编码会转换为 hashtable。

intset 编码的优势在于内存紧凑。例如，存储 100 个 int16_t 范围内的整数，intset 只需要 4（encoding）+ 4（length）+ 100 × 2 = 208 字节，
而使用 hashtable 则需要为每个 dictEntry 分配至少 24 字节（key 指针 + val 联合体 + next 指针），再加上哈希桶数组的开销，总内存消耗远高于 intset。


# 7 总结

Redis 的6种底层数据结构各有特点，它们的核心对比如下：

| 数据结构 | 时间复杂度（查找） | 时间复杂度（插入/删除） | 内存效率 | 主要应用场景 |
|---|---|---|---|---|
| **SDS** | O(1) 获取长度 | O(N) 追加（均摊 O(1)） | 预分配 + 惰性释放 | String 类型底层实现 |
| **ziplist/listpack** | O(N) 遍历 | O(N) | 极高（紧凑连续内存） | 小规模 Hash、ZSet、List |
| **quicklist** | O(N) | O(1)~O(N) | 较高 | List 类型底层实现 |
| **双向链表** | O(N) | O(1) | 较低（指针开销） | 发布订阅、慢查询等内部功能 |
| **哈希表** | O(1) | O(1) | 中等（哈希桶 + 链表开销） | 全局键空间、大规模 Hash/Set |
| **跳表** | O(logN) | O(logN) | 中等（多层指针） | Sorted Set（有序集合） |
| **整数集合** | O(logN) 二分查找 | O(N) 元素移动 | 极高（紧凑数组） | 纯整数小规模 Set |

Redis 之所以设计了这么多底层数据结构，核心目标是在**不同数据规模**下都能取得最优的时间和空间性能。小规模数据使用紧凑结构（ziplist、
listpack、intset）节省内存，大规模数据切换到高效结构（哈希表、跳表）保证操作速度。这种**编码转换**策略是 Redis 高性能的关键设计之一。
