
---
GMP模型
---

# 1 goroutine与线程
可增长的栈
OS线程（操作系统线程）一般都有固定的栈内存（通常为2MB）,一个goroutine的栈在其生命周期开始时只有很小的栈（典型情况下2KB），
goroutine的栈不是固定的，他可以按需增大和缩小，goroutine的栈大小限制可以达到1GB，虽然极少会用到这么大。所以在Go语言中
一次创建十万左右的goroutine也是可以的。

# 2 goroutine调度
GPM是Go语言运行时（runtime）层面的实现，是go语言自己实现的一套调度系统。区别于操作系统调度OS线程。
G很好理解，就是goroutine，里面除了存放本goroutine信息外，还有与所在P的绑定等信息。
P代表逻辑处理器（Logical Processor），用于调度 Goroutine 到 M 上运行。
P 维护一个本地的 Goroutine 队列，存储待执行的 Goroutine。
一个 M 必须绑定一个 P 才能执行 Goroutine。
Go 程序启动时，P 的数量由 GOMAXPROCS 决定，默认等于机器的 CPU 核心数。

P管理着一组goroutine队列，P里面会存储当前goroutine运行的上下文环境（函数指针，堆栈地址及地址边界），P会对自己管理的
goroutine队列做一些调度。
（比如把占用CPU时间较长的goroutine暂停、运行后续的goroutine等等）当自己的队列消费完了就去全局队列里取，如果全局队列里也
消费完了会去其他P的队列里抢任务(每次偷取一半)。
M（machine）是Go运行时（runtime）对操作系统内核线程的虚拟， M与内核线程一般是一一映射的关系， 一个goroutine最终是要放到
M上执行的；M 是 Goroutine 执行的载体，负责真正与操作系统交互。 
每个 M 绑定一个内核线程（Kernel Thread）。
M 负责运行绑定的 Goroutine，但一个 M 在某一时刻只能运行一个 Goroutine。
P与M一般也是一一对应的。他们关系是：P管理着一组G挂载在一个M上运行。当一个G长久阻塞在一个M上时，runtime会新建一个M，阻塞G所在
的P会把其他的G挂载在新建的M上。
当旧的G阻塞完成或者认为其已经死掉时回收旧的M。

# 3 GMP模型的运作
## 3.1 创建 Goroutine：
当创建一个新的 Goroutine 时（如通过 go func()），这个 Goroutine 会被添加到某个 P 的本地队列中。

## 3.2 调度循环：
调度器（Scheduler）会按照以下步骤工作：
从 P 的本地队列中取出一个 Goroutine，分配给绑定的 M 执行。
如果本地队列为空，P 会尝试从全局队列去取，如果全局队列也为空，会去其他 P 的队列中偷取 Goroutine（Work Stealing）。
如果找不到 Goroutine，M 会进入空闲状态。

### 3.2.1 M 和 P 的绑定：
M 在执行 Goroutine 时必须绑定一个 P。
如果 M 需要执行 Goroutine 而没有空闲的 P，会阻塞等待。

### 3.2.2 系统调用处理：
如果 Goroutine 执行了耗时的系统调用（如 I/O），M 会被阻塞。
为了避免整个 P 被阻塞：
调度器会分配一个新的 M 与 P 绑定，继续处理其他 Goroutine。
原来的 M 在系统调用完成后重新加入调度器。

### 3.2.3 Goroutine 的休眠与唤醒：
Goroutine 可能因为通道操作（chan）、锁等待、定时器（time.Sleep）等进入休眠状态。
调度器会将休眠的 Goroutine 暂存，直到其被唤醒后重新加入 P 的队列。

## 3.3 GMP 模型的特点
**轻量级 Goroutine**
相比线程，Goroutine 消耗的资源非常少。
Goroutine 的栈初始大小仅为 2KB，线程通常需要 1MB。

**动态扩展栈空间**
Goroutine 的栈可以根据需要动态扩展，最大支持到 1GB。

**Work Stealing（工作窃取)**
当一个 P 的任务队列为空时，它会尝试从其他 P 的任务队列中窃取任务，从而最大化 CPU 利用率。

**高效调度**
GMP 模型通过 P 来限制并发的 Goroutine 数量，避免了过多的线程切换。



# 4 GMP调度的优势
单从线程调度讲，Go语言相比起其他语言的优势在于OS线程是由OS内核来调度的，goroutine则是由Go运行时（runtime）自己的调度器
调度的，这个调度器使用一个称为m:n调度的技术（复用/调度m个goroutine到n个OS线程）。 其一大特点是goroutine的调度是在用户态
下完成的，不涉及内核态与用户态之间的频繁切换，包括内存的分配与释放，都是在用户态维护着一块大的内存池，不直接调用系统的
malloc函数（除非内存池需要改变），成本比调度OS线程低很多。
另一方面充分利用了多核的硬件资源，近似的把若干goroutine均分在物理线程上，再加上本身goroutine的超轻量，以上种种保证了
go调度方面的性能。

GOMAXPROCS
Go运行时的调度器使用GOMAXPROCS参数来确定需要使用多少个OS线程来同时执行Go代码。默认值是机器上的CPU核心数。例如在一个8核心的
机器上，调度器会把Go代码同时调度到8个OS线程上（GOMAXPROCS是m:n调度中的n）。
Go语言中可以通过runtime.GOMAXPROCS()函数设置当前程序并发时占用的CPU逻辑核心数。
Go1.5版本之前，默认使用的是单核心执行。Go1.5版本之后，默认使用全部的CPU逻辑核心数。

**m:n调度模型**

## 4.1  m:n 调度模型的工作原理
Goroutine 的创建：
用户代码可以通过 go 关键字创建 Goroutine，它们会被添加到某个 P 的任务队列。

Goroutine 的调度：
Goroutine 不直接与操作系统线程绑定，而是通过 P 和 M 的协作进行调度：
P（Processor）： 逻辑处理器，维护一个本地任务队列。
M（Machine）： 内核线程，负责实际运行 Goroutine。

映射关系：
每个 M 必须绑定一个 P 才能运行 Goroutine。
一个 P 可以将任务分配给多个 M，从而将 Goroutine 映射到多个线程上运行。
用户态切换：
Goroutine 的切换在用户态完成，避免了内核态的高开销。

## 4.2 m:n 调度的关键机制

### 4.2.1 Goroutine 的挂起与唤醒
当一个 Goroutine 因为 I/O 或系统调用阻塞时，M 会被阻塞。
调度器会分配一个新的 M 继续执行 P 上的其他 Goroutine，从而避免 P 被完全阻塞。

### 4.2.2 工作窃取（Work Stealing）
如果一个 P 的任务队列为空，它会尝试从其他 P 的任务队列中窃取任务，确保负载均衡。
全局队列（Global Queue）
除了每个 P 的本地队列，调度器还维护了一个全局队列。
如果所有 P 的任务队列都为空，P 会从全局队列中获取任务。

## 4.3 m:n 调度的优势
轻量级并发：
Goroutine 是用户态的执行单元，创建、销毁和切换的开销比操作系统线程小很多。

高效资源利用：
调度器动态分配 M 到 Goroutine，避免线程闲置，提高 CPU 使用率。

避免阻塞：
调度器通过分配新 M 处理阻塞 Goroutine，避免 P 被完全阻塞。

负载均衡：
通过工作窃取和全局队列，均衡了多个 P 的负载。


## 4.4 m:n 调度的局限性
锁竞争问题：
多个 P 同时访问全局队列可能引发锁竞争。

垃圾回收开销：
大量 Goroutine 存在时，垃圾回收的开销可能增大。

复杂性：
m:n 模型比 1:1 或 n:1 模型更复杂，调度器需要额外的逻辑来处理边界情况。
