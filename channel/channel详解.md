
---
channel详解
---

1 初始化
声明和初始化管道的方式主要有以下两种:
变量声明
使用内置函数make()

1> 变量声明

```golang
var ch chan int
```
这种方式声明的管道，值为nil。每个管道只能存储一种类型的数据。

2> 使用内置函数make()
使用内置函数make()可以创建无缓冲管道和有缓冲管道

```golang
c1 := make(chan int)
c2 := make(chan int, 5)
```

2 管道操作

1> 操作符
操作符<- 表示数据流向，管道在左表示向管道写入数据，管道在右表示从管道读出数据。
默认的管道为双向可读写，管道在函数间传递时可以使用操作符限制管道的读写，如下所示:

```golang
func NormalChan (ch chan int) {
	// 管道可读写
}

func ReadChan (ch <-chan int) {
	// 只能从管道读取数据
}

func WriteChan (ch chan<- int) {
    // 只能向管道写入数据
}
```

2> 数据读写
管道没有缓冲区时，从管道读取数据会阻塞，直到有协程向管道发送(写入)数据。类似的，向管道写入数据也会阻塞，直到有协程从
管道读取数据。

管道有缓冲区但缓冲区没有数据时，从管道读取数据也会阻塞，直到有协程写入数据。类似的，向管道写入数据，如果缓冲区已满，那么也会
阻塞，直到有协程从缓冲区中读出数据。
对于值为nil的管道，无论读写都会阻塞，而且是永久阻塞。
使用内置函数close()可以关闭管道，尝试向已经关闭的管道写入数据会触发panic，但关闭的管道仍然可读。

![channel-operation.png](images%2Fchannel-operation.png)

管道读取表达式最多可以给两个变量赋值

```golang
v := <-ch
v, ok := <-ch
```
第一个变量表示读出的数据，第二个变量(bool类型)表示是否成功读取了数据，需要注意的是，第二个变量不用于指示管道的关闭状态。
第二个变量常常会被误以为管道的关闭状态，其实它指示的是管道的缓冲区是否还有数据可读。

一个已关闭的管道有两种情况:
管道缓冲区已无数据；
管道缓冲区还有数据。

对于第一种情况，管道已关闭且缓冲区没有数据，那么管道读取表达式返回的第一个变量为相应类型的零值，第二个变量为false；
对于第一种情况，管道已关闭且缓冲区还有数据，那么管道读取表达式返回的第一个变量为读取到的数据，第二个变量为true。

3 小结
内置函数len() 和cap() 作用于管道，分别用于查询缓冲区中数据的个数及缓冲区的大小。
管道实现了一种FIFO，也就是先进先出的队列，数据总是按照写入的顺序流出管道。

协程读取时，阻塞的场景有:
管道为nil；
管道无缓冲区
管道有缓冲区但缓冲区无数据。

协程写入时，阻塞的场景有:
管道为nil；
管道无缓冲区
管道有缓冲区但缓冲区已满。

实现原理

1 数据结构


```golang
// src/runtime/chan.go hchan定义了管道的数据结构
type hchan struct {
   // 当前队列中剩余的元素个数
   qcount   uint
   // 环形队列长度，即可以存放的元素个数
   dataqsiz uint
   // 指向底层循环数组的指针
   // 只针对有缓冲的 channel
   buf      unsafe.Pointer
   // chan 中每个元素的大小
   elemsize uint16
   // chan 是否被关闭的标志
   closed   uint32
   // chan 中元素类型
   elemtype *_type // element type
   // 已发送元素在循环数组中的索引
   sendx    uint   // send index
   // 已接收元素在循环数组中的索引
   recvx    uint   // receive index
   // 等待接收的 goroutine 队列
   recvq    waitq  // list of recv waiters
   // 等待发送的 goroutine 队列
   sendq    waitq  // list of send waiters

   // 互斥锁，chan不允许并发读写
   lock mutex
}
```

从数据结构可以看出管道由队列、类型信息、协程等待队列完成。

1> 环形队列
chan内部实现了一个环形队列作为其缓冲区，队列的长度是在创建chan时指定的。
下图展示了一个可缓存6个元素的管道。

![hchan.png](images%2Fhchan.png)

dataqsiz指示了队列长度为6，即可以缓存6个元素；
buf指向队列的内存；
qcount表示队列中还有两个元素；
sendx指示后续写入的数据存储的位置，这里为4；
recvx指示后续从该位置读取数据，这里为0。

使用数组实现队列是比较常见的操作，sendx和recvx分别表示队尾和队首，分别指示数据数据写入的位置和数据读取的位置。

2> 等待队列
从管道读取数据时，如果管道缓冲区为空或者没有缓冲区时，则读取数据的协程会被阻塞，并被加入到recvq队列。向管道写入
数据时，如果管道缓冲区已满或者没有缓冲区时，则写入数据的协程会被阻塞，并被加入到sendq队列。

处于等待队列中的协程会在其他协程操作管道时被唤醒；
因为读取数据被阻塞的协程会被向管道写入数据的协程唤醒；
因为写入数据被阻塞的协程会被从管道读取数据的协程唤醒。

一般情况下，recvq和sendq至少有一个为空。只有一个例外，那就是同一个协程使用select语句向管道一边写入数据，一边读取数据，
此时协程会分别位于两个等待队列中。

3> 类型信息
一个管道只能传递一种类型的值，类型信息存储在hchan数据结构中。
elemtype代表类型，用于在数据传递过程中赋值；
elemsize代表类型大小，用于在buf中定位元素的位置。
如果需要管道传递任意类型的数据，则可以使用interfac{}类型。

4> 互斥锁
一个管道同时仅允许被一个协程读写。

2 管道操作

1> 创建管道
创建管道的过程实际上是初始化hchan结构，其中类型信息和缓冲区长度由内置函数make()指定，buf的大小则由元素大小和缓冲区容量共同决定。

2> 向管道写数据
如果缓冲区没满，则将数据写入缓冲区，结束发送过程。
如果缓冲区已满，则将当前协程加入sendq队列，进入睡眠并等待被读协程唤醒。

在实现时有一个小技巧，当接收队列recvq不为空时，说明缓冲区没有数据但有协程在等待数据，此时会把数据直接传递给recvq队列中的
第一个协程，而不必再写入缓冲区。

3> 从管道读数据
如果缓冲区有数据，则从缓冲区中取出数据，结束读取过程；
如果缓冲区中没有数据，则将当前协程加入到recvq队列，进入睡眠并等待被写协程唤醒。

类似地，如果等待发送队列sendq不为空，且没有缓冲区，那么此时将直接从sendq队列的第一个协程中获取数据。

4> 关闭管道
关闭管道时会把recvq中的协程全部唤醒，这些协程获取到的数据都是对应类型的零值。同时会把sendq队列中的协程全部唤醒，
但这些协程会触发panic。

除此之外，其他会触发panic的操作还有：
关闭一个已经关闭的管道；
关闭值为nil的管道；
向已经关闭的管道写入数据。

1> select
使用select可以监控多个管道，当其中某一个管道可操作时就触发相应的case分支。
如果多个管道都可操作时，会随机选出一个来读取。
尽管管道中没有数据，select的case语句读管道时不也会阻塞，这是因为case语句编译后调用读管道时会明确传入不阻塞参数，
读不到数据时不会将当前协程加入recvq等待队列，而是直接返回。

2> for-range
通过for-range可以持续地从管道中读取数据，好像在遍历一个数组一样，当管道中没有数据时会阻塞当前协程，与读管道时的阻塞
处理机制一样。即便管道被关闭，for-range也可以优雅地结束。



5 管道发送和接收元素的本质是什么?

管道 发送和接收元素的本质是什么？

> All transfer of value on the go channels happens with the copy of value.

就是说管道的发送和接收操作本质上都是 “值的拷贝”，无论是从 sender goroutine 的栈到 chan buf，还是从 
chan buf 到 receiver goroutine，或者是直接从 sender goroutine 到 receiver goroutine。

6 管道在什么情况下会引起资源泄漏？

Channel可能会引发 goroutine 泄漏。

泄漏的原因是 goroutine 操作 channel 后，处于发送或接收阻塞状态，而 channel 处于满或空的状态，一直得不到改变。同时，
垃圾回收器也不会回收此类资源，进而导致 goroutine 会一直处于等待队列中，不见天日。

另外，程序运行过程中，对于一个 channel，如果没有任何 goroutine 引用了，即便是它没有被关闭掉，gc 也会对其进行回收操作，
不会引起内存泄漏。

7  管道有哪些常用的应用场景

1>停止信号

参见close-chan-safely，这块就略过了。
channel用于停止信号的场景还是挺多的，经常是关闭某个channel或者向channel发送一个元素，使得接收channel的那一方获知道
此信息，进而做一些其他的操作。

2>任务定时
与 timer 结合，一般有两种玩法：实现超时控制，实现定期执行某个任务。
有时候，需要执行某项操作，但又不想它耗费太长时间，上一个定时器就可以搞定：

```golang
select {
   case <- time.After(100 * time.Millisecond):
   case <- s.stopc:
      return false
}
```

等待 100 ms 后，如果 s.stopc 还没有读出数据或者被关闭，就直接结束。这是来自 etcd 源码里的一个例子，这样的写法随处可见。
定时执行某个任务，也比较简单：

```golang
func worker() {
   ticker := time.Tick(1 * time.Second)
   for {
      select {
      case <- ticker:
         // 执行定时任务
         fmt.Println("执行 1s 定时任务")
      }
   }
}
```

每隔 1 秒种，执行一次定时任务。

3> 解耦生产者和消费者
服务启动时，启动 n 个 worker，作为工作协程池，这些协程工作在一个 for {} 无限循环里，从某个 channel 消费工作任务并执行：

```golang

package main

import (
	"fmt"
	"time"
)

func main() {
   taskCh := make(chan int, 100)
   go worker(taskCh)

    // 塞任务
   for i := 0; i < 10; i++ {
      taskCh <- i
   }

    // 等待 1 小时 
   select {
   case <-time.After(time.Hour):
   }
}

func worker(taskCh <-chan int) {
   const N = 5
   // 启动 5 个工作协程
   for i := 0; i < N; i++ {
      go func(id int) {
         for {
            task := <- taskCh
            fmt.Printf("finish task: %d by worker %d\n", task, id)
            time.Sleep(time.Second)
         }
      }(i)
   }
}
```

程序输出:
```shell
finish task: 1 by worker 4
finish task: 2 by worker 2
finish task: 4 by worker 3
finish task: 3 by worker 1
finish task: 0 by worker 0
finish task: 6 by worker 0
finish task: 8 by worker 3
finish task: 9 by worker 1
finish task: 7 by worker 4
finish task: 5 by worker 2
```

4  控制并发数
有时需要定时执行几百个任务，例如每天定时按城市来执行一些离线计算的任务。但是并发数又不能太高，因为任务执行过程依赖第三方
的一些资源，对请求的速率有限制。这时就可以通过 channel 来控制并发数。

下面的例子来自《Go 语言高级编程》：

```golang
var limit = make(chan int, 3)

func main() {
    // …………
    for _, w := range work {
        go func() {
            limit <- 1
            w()
            <-limit
        }()
    }
    // …………
}
```

构建一个缓冲型的 channel，容量为 3。接着遍历任务列表，每个任务启动一个 goroutine 去完成。真正执行任务，访问第三方的动作在 w() 中完成，在执行 w() 之前，先要从 limit 中拿“许可证”，拿到许可证之后，才能执行 w()，并且在执行完任务，要将“许可证”归还。这样就可以控制同时运行的 goroutine 数。

这里，limit <- 1 放在 func 内部而不是外部，原因是：

> 如果在外层，就是控制系统 goroutine 的数量，可能会阻塞 for 循环，影响业务逻辑。

>limit 其实和逻辑无关，只是性能调优，放在内层和外层的语义不太一样。

还有一点要注意的是，如果 w() 发生 panic，那“许可证”可能就还不回去了，因此需要使用 defer 来保证。