
---
Page Cache详解
---

# 1 如何用数据观测Page Cache?

Page Cache 是由内核管理的内存，它属于内核而不是用户。
Page Cache 存在的意义：减少 I/O，提升应用的 I/O 速度。





![proc-meminfo.png](images%2Fproc-meminfo.png)





根据上面的数据，你可以简单得出这样的公式：
Buffers + Cached + SwapCached = Active(file) + Inactive(file) + Shmem + SwapCached
那么等式两边的内容就是我们平时说的 Page Cache。请注意你没有看错，两边都有SwapCached，之所以要把它放在等式里，就是说它也是 Page Cache 的一部分。

在 Page Cache 中，Active(file)+Inactive(file) 是 File-backed page（与文件对应的内存页），是你最需要关注的部分。因为你平时用的 mmap() 
内存映射方式和 buffered I/O来消耗的内存就属于这部分，最重要的是，这部分在真实的生产环境上也最容易产生问题。

而 SwapCached 是在打开了 Swap 分区后，把 Inactive(anon)+Active(anon) 这两项里的匿名页给交换到磁盘（swap out），然后再读入到内存
（swap in）后分配的内存。由于读入到内存后原来的 Swap File 还在，所以 SwapCached 也可以认为是 File-backedpage，即属于 Page Cache。
这样做的目的也是为了减少 I/O。

SwapCached 只在 Swap 分区打开的情况下才会有，建议在生产环境中关闭Swap 分区，因为 Swap 过程产生的 I/O 会很容易引起性能抖动。
除了 SwapCached，Page Cache 中的 Shmem 是指匿名共享映射这种方式分配的内存（free 命令中 shared 这一项），比如 tmpfs
（临时文件系统），这部分在真实的生产环境中产生的问题比较少。


free 命令中的 buff/cache 究竟是指什么呢？
buff/cache = Buffers + Cached + SReclaimable
从这个公式中，你能看到 free 命令中的 buff/cache 是由 Buffers、Cached 和SReclaimable 这三项组成的，它强调的是内存的可回收性，
也就是说，可以被回收的内存会统计在这一项。
其中 SReclaimable 是指可以被回收的内核内存，包括 dentry 和 inode 等。

# 2 Page Cache是如何产生和释放的？

## 2.1 Page Cache的诞生

Page Cache 的产生有两种不同的方式：
这两种方式分别都是如何产生 Page Cache 的呢？来看下面这张图：
Page Cache产生方式示意图





![birth-of-page-cache.png](images%2Fbirth-of-page-cache.png)





从图中你可以看到，虽然二者都能产生 Page Cache，但是二者还是有些差异的：
标准 I/O 是写的 (write(2)) 用户缓冲区 (Userpace Page 对应的内存)，然后再将用户缓冲区里的数据拷贝到内核缓冲区
(Pagecache Page 对应的内存)；如果是读的 (read(2)) 话则是先从内核缓冲区拷贝到用户缓冲区，再从用户缓冲区读数据，
也就是 buffer 和文件内容不存在任何映射关系。
对于存储映射 I/O 而言，则是直接将 Pagecache Page 给映射到用户地址空间，用户直接读写 Pagecache Page 中内容。

Buffered I/O（标准 I/O）；
Memory-Mapped I/O（存储映射 I/O）。


显然，存储映射 I/O 要比标准 I/O 效率高一些，毕竟少了“用户空间到内核空间互相拷贝”的过程。这也是很多应用开发者发现，
为什么使用内存映射 I/O 比标准 I/O 方式性能要好一些的主要原因。



```shell
#! /bin/sh
# 这是我们用来解析的文件
MEM_FILE="/proc/meminfo"

# 这是在该脚本中将要生成的一个新文件
NEW_FILE="/home/python/dd.write.out"


# 我们用来解析page cache的具体项目

active=0
inactive=0
pagecache=0

IFS=' '

#从/proc/meminfo中读取file page cache的大小
function get_filecache_size()
{
        items=0
        while read line
        do
                if [[ "$line" =~ "Active:" ]];then
                        read -ra ADDR <<<"$line"
                        active=${ADDR[1]}
                        let items="$items+1"
                elif [[ "$line" =~ "Inactive:" ]];then
                        read -ra ADDR <<<"$line"
                        inactive=${ADDR[1]}
                        let items="$items+1"
                fi
                
                if [ $items -eq 2 ];then
                        break;
                fi
        done < $MEM_FILE
}

# 读取file page cache的初始大小
get_filecache_size
let filecache="$active + $inactive"

# 写一个新文件，该文件大小为366000kb
dd if=/dev/zero of=$NEW_FILE bs=1024 count=366000 &> /dev/null

# 文件写完后，再次读取file page cache的大小
get_filecache_size

# 两次的差异可以近似为该新文件内容对应的file page cache大小
# 之所以是近似，是因为在运行过程中可能会有其他page cache产生
let size_increased="$active + $inactive - $filecache"

# 输出结果
echo "File Size 366000KB, File Cache Increased" $size_increased 
```

在运行该脚本前你要确保系统中有足够多的 free 内存（避免内存紧张产生回收行为），最终的测试结果是这样的：





![test-page-cache.png](images%2Ftest-page-cache.png)





通过这个脚本你可以看到，在创建一个文件的过程中，代码中 /proc/meminfo 里的Active(file) 和 Inactive(file) 
这两项会随着文件内容的增加而增加，它们增加的大小跟文件大小是一致的（这里之所以略有不同，是因为系统中还有其他程序在运行）。
另外，如果你观察得很仔细的话，你会发现增加的 Page Cache 是 Inactive(File) 这一项，这是因为active(File) 和 
Inactive(File) 分别表示活跃的文件内存和非活跃的文件内存，前者是最近使用过的内存，通常不会被回收用作其他用途，
而后者是最近没有使用过的内存，可以被回收用作其他用途。本案例中，新文件dd.write.out 只写了一次，写完后会被读入内存，
但是后续没有被使用(读或写)，因此属于最近没有使用过的内存，所以属于非活跃的文件内存，也就是 Inactive(File)。

当然，这个过程看似简单，但是它涉及的内核机制还是很多的，换句话说，可能引起问题的地方还是很多的，我们用一张图简单描述
下这个过程：





![desc-process.png](images%2Fdesc-process.png)





这个过程大致可以描述为：首先往用户缓冲区 buffer(这是 Userspace Page) 写入数据，然后 buffer 中的数据拷贝到
内核缓冲区（这是 Pagecache Page），如果内核缓冲区中还没有这个 Page，就会发生 Page Fault ，也就是缺页中断，
会去分配一个 Page，拷贝结束后该 Pagecache Page 是一个 Dirty Page（脏页），然后该 Dirty Page 中的内容会
同步到磁盘，同步到磁盘后，该 Pagecache Page 变为 Clean Page 并且继续存在系统中。


我们可以将 Alloc Page 理解为 Page Cache 的“诞生”，将 Dirty Page 理解为Page Cache 的婴幼儿时期（最容易生病的时期），
将 Clean Page 理解为 Page Cache的成年时期（在这个时期就很少会生病了）。
但是请注意，并不是所有人都有童年的，比如孙悟空，一出生就是成人了，Page Cache也一样，如果是读文件产生的 Page Cache，
它的内容跟磁盘内容是一致的，所以它一开始是 Clean Page，除非改写了里面的内容才会变成 Dirty Page（返老还童）。

就像我们为了让婴幼儿健康成长，要悉心照料他 / 她一样，为了提前发现或者预防婴幼儿时期的 Page Cache 发病，
我们也需要一些手段来观测它：

```shell
$ cat /proc/vmstat | egrep "dirty|writeback"
nr_dirty 40
nr_writeback 2
```

如上所示，nr_dirty 表示当前系统中积压了多少脏页，nr_writeback 则表示有多少脏页正在回写到磁盘中，他们两个的单位
都是 Page(4KB)。通常情况下，小朋友们（Dirty Pages）聚集在一起（脏页积压）不会有什么问题，但在非常时期比如流感期间，
就很容易导致聚集的小朋友越多病症就会越严重。与此类似，Dirty Pages 如果积压得过多，在某些情况下也会容易引发问题。

# 2.2 Page Cache的死亡

我们可以把 Page Cache 的回收行为 (Page Reclaim) 理解为 Page Cache 的“自然死亡”。

free 命令中的 buff/cache 中的这些就是“活着”的 Page Cache，那它们什么时候会“死亡”（被回收）呢？我们来看一张图：





![death-of-page-cache.png](images%2Fdeath-of-page-cache.png)





你可以看到，应用在申请内存的时候，即使没有 free 内存，只要还有足够可回收的 PageCache，就可以通过回收 Page Cache 
的方式来申请到内存，回收的方式主要是两种：直接回收和后台回收。

那它是具体怎么回收的呢？你要怎么观察呢？观察 Page Cache 直接回收和后台回收最简单方便的方式是使用 sar





![sar-page-cache.png](images%2Fsar-page-cache.png)





借助上面这些指标，你可以更加明确地观察内存回收行为，下面是这些指标的具体含义：
pgscank/s : kswapd(后台回收线程) 每秒扫描的 page 个数。
pgscand/s: Application 在内存申请过程中每秒直接扫描的 page 个数。
pgsteal/s: 扫描的 page 中每秒被回收的个数。
%vmeff: pgsteal/(pgscank+pgscand), 回收效率，越接近 100 说明系统越安全，越接
近 0 说明系统内存压力越大。

Page Cache 是在应用程序读写文件的过程中产生的，所以在读写文件之前你需要留意是否还有足够的内存来分配 Page Cache；
Page Cache 中的脏页很容易引起问题，你要重点注意这一块；
在系统可用内存不足的时候就会回收 Page Cache 来释放出来内存，我建议你可以通过sar 或者 /proc/vmstat 来观察这个
行为从而更好的判断问题是否跟回收有关。

总的来说，Page Cache 的生命周期对于应用程序而言是相对比较透明的，即它的分配与回收都是由操作系统来进行管理的。
正是因为这种“透明”的特征，所以应用程序才会难以控制 Page Cache，Page Cache 才会容易引发那么多问题。



为什么第一次读写某个文件，Page Cache是 Inactive 的？如何让它变成 Active 的呢？在什么情况下 Active 的又会变成 Inactive的呢？
系统中有哪些控制项可以影响 Inactive与 Active Page Cache 的大小或者二者的比例？
对于匿名页而言，当产生一个匿名页后它会首先放在 Active 链表上；而对于文件页而言，当产生一个文件页后它会首先放在Inactive 链表上。
请问为什么会这样子？这是合理的吗？

在理解这些问题之前，需要先了解操作系统中的页缓存（Page Cache）概念。页缓存是操作系统中用于缓存磁盘上的数据页的一种机制，
以提高文件系统性能。当程序读取文件时，文件的内容会被缓存在页缓存中，以便后续读取时可以更快地获取数据。

> 为什么第一次读写某个文件，Page Cache 是 Inactive 的？如何让它变成 Active 的呢？

第一次读写某个文件时，操作系统会将该文件的页缓存放置在 Inactive 链表上，这是因为这些页缓存还没有被使用过，所以暂时被标记为
Inactive。当文件的页被访问时，操作系统会将这些页缓存移到 Active 链表上。要将一个页缓存从 Inactive 变为 Active，
只需简单地访问该页缓存，例如读取文件内容。

> 在什么情况下 Active 的又会变成 Inactive 的呢？

当操作系统发现系统内存不足时，会尝试将一些活跃的页缓存移动到 Inactive 状态，以便为新的页缓存腾出空间。这通常是通过操作系统的
页面置换算法来实现的，例如LRU（最近最少使用）算法。当某些页长时间未被访问或者系统内存紧张时，操作系统会将这些页缓存从 Active
状态移动到 Inactive 状态。

> 系统中有哪些控制项可以影响 Inactive 与 Active Page Cache 的大小或者二者的比例？

操作系统通常提供了一些调整页缓存管理行为的参数，例如 Linux 下的 vm.dirty_ratio、vm.dirty_background_ratio 等参数，
可以用来调整脏页（已被修改但尚未写回磁盘）的比例。调整这些参数可以间接地影响页缓存的大小和 Active 与 Inactive 的比例。

> 为什么会出现匿名页和文件页放置到不同的链表上？

匿名页和文件页在内存管理中具有不同的特性。匿名页通常包含了进程的私有数据，因此在使用过程中需要频繁读写，并且换出时需要写回
到交换空间，因此内核会优先放置在 Active 链表上，以提高访问速度和性能。相比之下，文件页通常包含了静态数据或可执行代码，
读写频率较低，所以放置在 Inactive 链表上不会对性能产生太大影响。

> inactive匿名内存为什么不可以被回收？

在操作系统中，匿名内存通常指的是进程使用的堆内存和栈内存等，这些内存空间是为进程动态分配的，通常存储着进程的私有数据。
与文件内存不同，匿名内存没有对应的存储介质，因此无法被回收到磁盘上，而是存在于进程的虚拟地址空间中。

当匿名内存处于 Inactive 状态时，虽然暂时没有被活跃地使用，但操作系统仍然需要保留这部分内存，因为它包含着进程的重要数据。
如果将匿名内存回收掉，进程可能会在之后的使用过程中出现内存访问错误或者数据丢失的情况。
另外，匿名内存的回收和释放通常由进程自身的内存管理机制来负责，而不是由操作系统直接进行。当进程不再需要某些匿名内存时，
它会通过释放内存的系统调用（例如 C 语言中的 free() 函数）来主动释放这部分内存，以便操作系统将其重新分配给其他进程使用。

因此，尽管匿名内存在一段时间内处于 Inactive 状态，但操作系统仍然需要保留它，并等待进程释放或重新激活它。


**什么是匿名页和文件页？**

匿名页背后并没有一个磁盘中的文件作为数据来源，匿名页中的数据都是通过进程运行过程中产生的，匿名页直接和进程虚拟地址空间建立映射供进程使用。
匿名页是指进程的私有内存页，通常用于存储进程的堆栈、堆分配的内存以及匿名映射的内存。
文件页中的数据来自于磁盘中的文件，文件页需要先关联一个磁盘中的文件，然后再和进程虚拟地址空间建立映射供进程使用，使得进程可以通过操作
虚拟内存实现对文件的操作，这就是我们常说的内存文件映射。
文件页则是从文件系统中映射到内存中的页面，通常用于存储程序的可执行代码和静态数据。

**两者的换出 Swap Out 成本不同**
写入内容不同：匿名页中的内容通常是进程的私有数据，例如堆栈数据或动态分配的数据，这些数据在运行时会经常发生变化。因此，当需要换出
匿名页时，必须将其中的数据写回到交换空间（Swap Space）中，以保证数据的完整性和一致性。相比之下，文件页中的内容通常是静态的，
不需要频繁地写回到交换空间。
写入量不同：由于匿名页通常包含进程的私有数据，它们可能包含大量的写入操作，这会导致需要频繁地将数据写回到交换空间中。而文件页
通常是只读的或者很少被写入的，因此写回的频率较低。
对进程性能的影响：由于匿名页包含了进程的私有数据，频繁地将匿名页换出可能会导致进程性能下降，因为每次需要访问被换出的匿名页时，
都需要从交换空间中将其读回到内存中，这会增加访问延迟。相比之下，文件页的换出对进程性能的影响相对较小，因为文件页通常不包含
进程的关键数据。


**为什么会有 active 链表和 inactive 链表？**

LRU 算法更多的是在时间维度上的考量，突出最近最少使用，但是它并没有考量到使用频率的影响，假设有这样一种状况，就是一个页面被疯狂频繁地使用，
毫无疑问它肯定是一个热页，但是这个页面最近的一次访问时间离现在稍微久了一点点，此时进来大量的页面，这些页面的特点是只会使用一两次，以后将再也不会用到。
在这种情况下，根据 LRU 的语义这个之前频繁地被疯狂访问的页面就会被置换出去了（本来应该将这些大量一次性访问的页面置换出去的），当这个页面
在不久之后要被访问时，此时已经不在内存中了，还需要在重新置换进来，造成性能的损耗。这种现象也叫 Page Thrashing（页面颠簸）。
因此，内核为了将页面使用频率这个重要的考量因素加入进来，于是就引入了 active 链表和 inactive 链表。工作原理如下：

1. 首先 inactive 链表的尾部存放的是访问频率最低并且最少访问的页面，在内存紧张的时候，这些页面被置换出去的优先级是最大的。 
2. 对于文件页来说，当它被第一次读取的时候，内核会将它放置在 inactive 链表的头部，如果它继续被访问，则会提升至 active 链表的尾部。
如果它没有继续被访问，则会随着新文件页的进入，内核会将它慢慢的推到 inactive 链表的尾部，如果此时再次被访问则会直接被提升到
active 链表的头部。大家可以看出此时页面的使用频率这个因素已经被考量了进来。
3. 对于匿名页来说，当它被第一次读取的时候，内核会直接将它放置在 active 链表的尾部，注意不是 inactive 链表的头部，
这里和文件页不同。因为匿名页的换出 Swap Out 成本会更大，内核会对匿名页更加优待。当匿名页再次被访问的时候就会被被提升
到 active 链表的头部。
4. 当遇到内存紧张的情况需要换页时，内核会从 active 链表的尾部开始扫描，将一定量的页面降级到 inactive 链表头部，
这样一来原来位于 inactive 链表尾部的页面就会被置换出去。
内核在回收内存的时候，这两个列表中的回收优先级为：inactive 链表尾部 > inactive 链表头部 > active 链表尾部 > active 链表头部。

# 3 如何处理Page Cache难以回收产生的load飙高问题？

在生产环境中，因为 PageCache 管理不当引起的系统负载飙高的问题主要是以下三种情况引发的：
> 直接内存回收引起的 load 飙高；
> 系统中脏页积压过多引起的 load 飙高；
> 系统 NUMA 策略配置不当引起的 load 飙高。

## 3.1 直接内存回收引起load飙高或者业务时延抖动

直接内存回收是指在进程上下文同步进行内存回收，那么直接内存回收具体是怎么引起load 飙高的呢？
因为直接内存回收是在进程申请内存的过程中同步进行的回收，而这个回收过程可能会消耗很多时间，进而导致进程的后续行为都被迫
等待，这样就会造成很长时间的延迟，以及系统的 CPU 利用率会升高，最终引起 load 飙高。
用一张图来描述这个过程就是：





![direct-mem-recycle.png](images%2Fdirect-mem-recycle.png)





从图里你可以看到，在开始内存回收后，首先进行后台异步回收（上图中蓝色标记的地方），这不会引起进程的延迟；如果后台
异步回收跟不上进行内存申请的速度，就会开始同步阻塞回收，导致延迟（上图中红色和粉色标记的地方，这就是引起 load 高的地方）。
那么，针对直接内存回收引起 load 飙高或者业务 RT 抖动的问题，一个解决方案就是及早地触发后台回收来避免应用程序进行
直接内存回收，那具体要怎么做呢？

我们先来了解一下后台回收的原理，如图：





![backend-recycle.png](images%2Fbackend-recycle.png)





它的意思是：当内存水位低于 watermark low 时，就会唤醒 kswapd 进行后台回收，然后 kswapd 会一直回收到 watermark high。

那么，我们可以增大 min_free_kbytes 这个配置选项来及早地触发后台回收，该选项最终控制的是内存回收水位。

```shell
vm.min_free_kbytes = 4194304
```

对于大于等于 128G 的系统而言，将 min_free_kbytes 设置为 4G 比较合理，这是处理很多这种问题时总结出来的一个经验值，
既不造成较多的内存浪费，又能避免掉绝大多数的直接内存回收。
该值的设置和总的物理内存并没有一个严格对应的关系，我们在前面也说过，如果配置不当会引起一些副作用，所以在调整该值之前，
我的建议是：你可以渐进式地增大该值，比如先调整为 1G，观察 sar -B 中 pgscand 是否还有不为 0 的情况；如果存在不为 0 的情况，
继续增加到 2G，再次观察是否还有不为 0 的情况来决定是否增大，依此类推。

这个方法可以用在 3.10.0 以后的内核上（对应的操作系统为 CentOS-7 以及之后更新的操作系统）。
当然了，这样做也有一些缺陷：提高了内存水位后，应用程序可以直接使用的内存量就会减少，这在一定程度上浪费了内存。所以在调整这
一项之前，你需要先思考一下，应用程序更加关注什么，如果关注延迟那就适当地增大该值，如果关注内存的使用量那就适当地调小该值。
总的来说，通过调整内存水位，在一定程度上保障了应用的内存申请，但是同时也带来了一定的内存浪费，因为系统始终要保障有这么多的
free 内存，这就压缩了 Page Cache 的空间。调整的效果你可以通过 /proc/zoneinfo 来观察：

```shell
egrep "min|low|high" /proc/zoneinfo
```





![proc-zoneinfo.png](images%2Fproc-zoneinfo.png)





其中 min、low、high 分别对应上图中的三个内存水位。你可以观察一下调整前后 min、low、high 的变化。
需要提醒你的是，内存水位是针对每个内存 zone 进行设置的，所以/proc/zoneinfo 里面会有很多 zone 以及它们的内存水位。


## 3.2 系统中脏页过多引起load飙高

前面提到过，在直接内存回收过程中，如果存在较多脏页就可能涉及在回收过程中进行回写，这可能会造成非常大的延迟，而且因为
这个过程本身是阻塞式的，所以又可能进一步导致系统中处于D 状态的进程数增多，最终的表现就是系统的 load 值很高。

我们来看一下这张图，这是一个典型的脏页引起系统 load 值飙高的问题场景：





![dirty-pages-cause-load-high.png](images%2Fdirty-pages-cause-load-high.png)





如图所示，如果系统中既有快速 I/O 设备，又有慢速 I/O 设备（比如图中的 ceph RBD 设备，或者其他慢速存储设备比如 HDD），
直接内存回收过程中遇到了正在往慢速 I/O 设备回写的 page，就可能导致非常大的延迟。

那如何解决这类问题呢？一个比较省事的解决方案是控制好系统中积压的脏页数据。很多人知道需要控制脏页，但是往往并不清楚如何
来控制好这个度，脏页控制的少了可能会影响系统整体的效率，脏页控制的多了还是会触发问题，所以我们接下来看下如何来衡量好这个“度”。

首先我们可以通过 sar -r 来观察系统中的脏页个数：





![sar-dirty-pages.png](images%2Fsar-dirty-pages.png)





kbdirty 就是系统中的脏页大小，它同样也是对 /proc/vmstat 中 nr_dirty 的解析。你可以通过调小如下设置来将系统脏页
个数控制在一个合理范围:

```shell
vm.dirty_background_bytes = 0
vm.dirty_background_ratio = 10
vm.dirty_bytes = 0
vm.dirty_expire_centisecs = 3000
vm.dirty_ratio = 20
```

调整这些配置项有利有弊，调大这些值会导致脏页的积压，但是同时也可能减少了 I/O 的次数，从而提升单次刷盘的效率；调小这些值可以
减少脏页的积压，但是同时也增加了I/O 的次数，降低了 I/O 的效率。

至于这些值调整大多少比较合适，也是因系统和业务的不同而异，我的建议也是一边调整一边观察，将这些值调整到业务可以容忍的
程度就可以了，即在调整后需要观察业务的服务质量 (SLA)，要确保 SLA 在可接受范围内。调整的效果你可以通过 /proc/vmstat 来查看：

```shell
grep "nr_dirty_" /proc/vmstat
nr_dirty_threshold 366998
nr_dirty_background_threshold 183275
```

你可以观察一下调整前后这两项的变化。


## 3.3 系统NUMA策略配置不当引起的load飙高

除了我前面提到的这两种引起系统 load 飙高或者业务延迟抖动的场景之外，还有另外一种场景也会引起 load 飙高，那就是系统
NUMA 策略配置不当引起的 load 飙高。
比如说，我们在生产环境上就曾经遇到这样的问题：系统中还有一半左右的 free 内存，但还是频频触发 direct reclaim，导致
业务抖动得比较厉害。后来经过排查发现是由于设置了zone_reclaim_mode，这是 NUMA 策略的一种。

当某个Node内存不足时，系统可以从其他Node寻找空闲内存，也可以从本地内存中回收内存，具体选择哪种模式，可以通过
/proc/sys/vm/zone_reclaim_mode来调整，它支持以下几个选项:
默认为0，也就是既可以从其他Node寻找空闲内存，也可以从本地回收内存。
1、2、4 都表示只回收本地内存，2表示可以回写脏数据回收内存，4表示可以用Swap方式回收内存。

推荐使用默认配置0，设置 zone_reclaim_mode 的目的是为了增加业务的 NUMA 亲和性，但是在实际生产环
境中很少会有对 NUMA 特别敏感的业务。配置为 0 之后，就避免了在其他 node 有空闲内存时，不去使用这些空闲内存而是
去回收当前 node 的 Page Cache，也就是说，通过减少内存回收发生的可能性从而避免它引发的业务延迟。

我们可以通过 numactl 来查看服务器的 NUMA 信息:





![numactl.png](images%2Fnumactl.png)





推荐将 zone_reclaim_mode 配置为 0。
vm.zone_reclaim_mode = 0
因为相比内存回收的危害而言，NUMA 带来的性能提升几乎可以忽略，所以配置为 0，利远大于弊。


# 4 如何处理Page Cache容易回收引起的业务性能问题？

此类问题大致可以分为两方面：
> 误操作导致Page Cache被回收掉，进而导致业务性能下降明显。
> 内核的一些机制导致业务Page Cache被回收，从而引起性能下降。

如果你的业务对Page Cache比较敏感，比如说业务数据对延迟很敏感，或者再具体一点，你的业务指标对TR99(99分位)要求较高，
那你对于这类性能问题应该多多少少有所接触。

## 4.1 对Page Cache操作不当产生的业务性能下降

我们知道，对于Page Cache而言，是可以通过drop_cache来清理的，很多人在看到系统中存在非常多的Page Cache时会习惯性
地使用drop_cache来清理它们，但是这样做是会有一些负面影响的，比如说这些Page Cache被清理掉后可能会引起系统性能下降，为什么？
其实这与inode有关，inode是内存中对磁盘文件的索引，进程在查找或读取文件时就是通过inode来进行操作的，如下图所示：





![inode.png](images%2Finode.png)





如上图所示，进程会通过inode来找到文件的地址空间(adress_space)，然后结合文件偏移(会转换成page index)来找具体的
Page。如果该Page存在，那就说明文件内容已经被读取到了内存；否则就说明不在内存中，需要到磁盘中去读取。可以理解为inode
是Page Cache Page(页缓存的页)的宿主，如果inode不存在了，那么Page Cache Page也就不存在了。

我们知道drop_cache有几个控制选项，我们可以通过写入不同的数值来释放不同类型的cache(用户数据Page Cache，内核数据Slab，或者二者都释放)。





![drop-cache.png](images%2Fdrop-cache.png)





于是这样就引入了一个容易被我们忽略的问题: 当我们执行echo 2 > /proc/sys/vm/drop_caches 来释放Slab时，它也会把Page Cache给释放掉，
很多运维人员都会忽略掉这一点。而这会导致Page Cache Page(页缓存的页)的宿主inode被释放掉，进而导致Page Cache Page被清理掉，
最终引发业务性能的明显下降。

那么，有没有办法来观察这个inode释放引起Page Cache被释放的行为呢？答案是有的。

由于drop_caches是一种内存事件，内核会在/proc/vmstat中来记录这一事件，所以我们可以通过/proc/vmstat来判断是否有执行过drop_caches。





![grep-drop-caches.png](images%2Fgrep-drop-caches.png)





如上所示，它们分别意味着page cache被drop了6次(通过echo 1或echo 3)，slab被drop了6次(通过echo 2或者echo 3)。
如果这两个值在问题发生前后没有变化，那就可以排除是有人执行了drop_caches；否则可以认为是因为drop_caches引起的page cache被回收。


## 4.2 内核机制引起Page Cache被回收而产生的业务性能下降

我们知道，在内存紧张时会触发内存回收，内存回收会尝试去回收reclaimable（可以被回收的）的内存，这部分内存既包含Page Cache又包含
reclaimable kernel memory（比如slab）。我们可以用下图来简单描述这个过程:





![mem-recycle-process.png](images%2Fmem-recycle-process.png)





如上图所示，Reclaimer是回收者，它可以是内核线程 （包括kswapd），也可以是用户线程。回收时，它会依次来扫描
page cache page和slab page中有哪些是可以被回收的，如果有的话就会去尝试回收，如果没有的话就跳过。在扫描
可回收page的过程中回收者一开始扫描的较少，然后逐渐增加扫描比例直至全部都被扫描完。这就是内存回收的大致过程。

前面提到，如果inode被回收的话，那么其对应的Page Cache也都会被回收掉，所以如果业务进程读取的文件对应的inode
被回收了，那么该文件所有的Page Cache都会被释放掉，这也是容易引起性能问题的地方。

那这个行为是否有办法观察呢？这同样也是可以通过/proc/vmstat来观察的。





![grep-vmstat.png](images%2Fgrep-vmstat.png)





这个行为对应的事件是inodesteal，就是上面两个事件，其中kswapd_inodesteal是指在kswapd回收过程中，因为回收inode
而释放的page cache个数；pginodesteal是指kswapd之外其他线程在回收过程中，因为回收inode而释放的page cache page
个数。所以在你发现业务的Page Cache被释放掉后，你可以通过观察来发现是否是因为该事件导致的。


## 4.3 如何避免Page Cache被回收而引发的性能问题？

避免Page Cache里相对比较重要的数据被回收掉的思路有两种:
从应用代码层面来优化；
从系统层面来调整。

从应用程序代码层面来解决是相对比较彻底的方案，因为应用更清楚哪些Page Cache是重要的，哪些是不重要的，所以就可以明确地
来对读写过程中产生的Page Cache区别对待。比如说，对于重要的数据，可以通过mlock(2)来保护它，防止被回收以及被drop；对于
不重要的数据(比如日志)，那可以通过madvise(2)告诉内核来立即释放这些Page Cache。

确认Page Cache是否被保护住了，被保护了多大，同样可以通过/proc/meminfo来观察:





![grep-meminfo.png](images%2Fgrep-meminfo.png)





然后你可以发现，drop_caches或者内存回收是回收不了这些内容的，我们的目的也就达到了。

在有些情况下，对应用程序而言，修改源码是比较麻烦的事，如果可以不修改源码来达到目的自然最好不过。Linux内核同样实现了
这种不改应用程序的源码而从系统层面调整来保护重要数据的机制，这个机制就是memory cgroup protection。

它大致的思路是，将需要保护的应用程序使用memory cgroup来保护起来，这样该应用程序读写文件过程中所产生的Page Cache
就会被保护起来不被回收或者最后被回收，memory cgroup protection大致的原理如下图所示:





![mem-cgroup-protection.png](images%2Fmem-cgroup-protection.png)





如上图所示，memory cgroup提供了几个内存水位控制线memory.{min、low、high、max}。

memory.max
这是指memory cgroup内的进程最多能够分配的内存，如果不设置的话，就默认不做内存大小的限制。

memory.high
如果设置了这一项，当memory cgroup内进程的内存使用量超过了该值后就会立即被回收掉，所以这一项的目的是为了尽快回收
掉不活跃的Page Cache。

memory.low
这一项是用来保护重要数据的，当memory cgroup内进程的内存使用量低于该值后，在内存紧张触发回收后就会先去回收不属于
该memory cgroup的Page Cache，等到其他的Page Cache都被回收后才会去回收这些Page Cache。

memory.min
这一项也是用来保护重要数据的，只不过与memory.low有所不同的是，当memory cgroup内进程的内存使用量低于该值后，
即使其他不在该memory cgroup内的Page Cache都被回收完了也不会去回收这些Page Cache，可以理解为这是用来保护
最高优先级的数据的。

所以，如果你想要保护你的Page Cache不被回收，你就可以考虑将你的业务进程放在一个memory cgroup中，然后设置
memory.{min，low}来进行保护；反之，如果你想要尽快释放掉你的Page Cache，那你可以考虑设置memory.high来
及时的释放掉不活跃的Page Cache。