
---
channel详解
---

**不要通过共享内存来通信，而应该通过通信来共享内存**

# 1 初始化
声明和初始化管道的方式主要有以下两种:
变量声明
使用内置函数make()

## 1.1 变量声明

```golang
var ch chan int
```
这种方式声明的管道，值为nil。每个管道只能存储一种类型的数据。

## 1.2 使用内置函数make()
使用内置函数make()可以创建无缓冲管道和有缓冲管道

```golang
c1 := make(chan int)
c2 := make(chan int, 5)
```

# 2 管道操作

## 2.1 操作符
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

## 2.2 数据读写
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
对于第二种情况，管道已关闭且缓冲区还有数据，那么管道读取表达式返回的第一个变量为读取到的数据，第二个变量为true。

## 2.3 小结
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

# 3 实现原理

## 3.1 数据结构


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

### 3.1.1 环形队列
chan内部实现了一个环形队列作为其缓冲区，队列的长度是在创建chan时指定的。
下图展示了一个可缓存6个元素的管道。

![hchan.png](images%2Fhchan.png)

dataqsiz指示了队列长度为6，即可以缓存6个元素；
buf指向队列的内存；
qcount表示队列中还有4个元素；
sendx指示后续写入的数据存储的位置，这里为4；
recvx指示后续从该位置读取数据，这里为0。

使用数组实现队列是比较常见的操作，sendx和recvx分别表示队尾和队首，分别指示数据数据写入的位置和数据读取的位置。

### 3.1.2 等待队列
从管道读取数据时，如果管道缓冲区为空或者没有缓冲区时，则读取数据的协程会被阻塞，并被加入到recvq队列。向管道写入
数据时，如果管道缓冲区已满或者没有缓冲区时，则写入数据的协程会被阻塞，并被加入到sendq队列。

处于等待队列中的协程会在其他协程操作管道时被唤醒；
因为读取数据被阻塞的协程会被向管道写入数据的协程唤醒；
因为写入数据被阻塞的协程会被从管道读取数据的协程唤醒。

一般情况下，recvq和sendq至少有一个为空。只有一个例外，那就是同一个协程使用select语句向管道一边写入数据，一边读取数据，
此时协程会分别位于两个等待队列中。

### 3.1.3 类型信息
一个管道只能传递一种类型的值，类型信息存储在hchan数据结构中。
elemtype代表类型，用于在数据传递过程中赋值；
elemsize代表类型大小，用于在buf中定位元素的位置。
如果需要管道传递任意类型的数据，则可以使用interface{}类型。

### 3.1.4 互斥锁
一个管道同时仅允许被一个协程读写。

## 3.2 管道操作

### 3.2.1 创建管道
创建管道的过程实际上是初始化hchan结构，其中类型信息和缓冲区长度由内置函数make()指定，buf的大小则由元素大小和缓冲区容量共同决定。

### 3.2.2 向管道写数据
如果缓冲区没满，则将数据写入缓冲区，结束发送过程。
如果缓冲区已满，则将当前协程加入sendq队列，进入睡眠并等待被读协程唤醒。

在实现时有一个小技巧，当接收队列recvq不为空时，说明缓冲区没有数据但有协程在等待数据，此时会把数据直接传递给recvq队列中的
第一个协程，而不必再写入缓冲区。

### 3.2.3 从管道读数据
如果缓冲区有数据，则从缓冲区中取出数据，结束读取过程；
如果缓冲区中没有数据，则将当前协程加入到recvq队列，进入睡眠并等待被写协程唤醒。

类似地，如果等待发送队列sendq不为空，且没有缓冲区，那么此时将直接从sendq队列的第一个协程中获取数据。

### 3.2.4 关闭管道
关闭管道时会把recvq中的协程全部唤醒，这些协程获取到的数据都是对应类型的零值。同时会把sendq队列中的协程全部唤醒，
但这些协程会触发panic。

除此之外，其他会触发panic的操作还有：
关闭一个已经关闭的管道；
关闭值为nil的管道；
向已经关闭的管道写入数据。

### 3.2.5 select
使用select可以监控多个管道，当其中某一个管道可操作时就触发相应的case分支。
如果多个管道都可操作时，会随机选出一个来读取。
尽管管道中没有数据，select的case语句读管道时也不会阻塞，这是因为case语句编译后调用读管道时会明确传入不阻塞参数，
读不到数据时不会将当前协程加入recvq等待队列，而是直接返回。

### 3.2.6 for-range
通过for-range可以持续地从管道中读取数据，好像在遍历一个数组一样，当管道中没有数据时会阻塞当前协程，与读管道时的阻塞
处理机制一样。即便管道被关闭，for-range也可以优雅地结束。
for-range 会阻塞等待管道中的数据。
只有管道被关闭，for-range 才会优雅地结束循环。
因此，在使用 for-range 遍历管道时，务必保证生产者 goroutine 在合适时机关闭管道，否则会触发死锁，比如下面这个案例：

```go
func main() {
	ch := make(chan int)

	go func() {
		for i := 1; i <= 5; i++ {
			ch <- i
		}
		// 注意：此处未关闭通道！
	}()

	for val := range ch {
		fmt.Println(val)
	}
}

执行代码会panic: fatal error: all goroutines are asleep - deadlock!

```

正确操作方式
```go
package main

import (
	"fmt"
	"time"
)

func main() {
	ch := make(chan int)

	// 开启一个 goroutine 写入数据并关闭通道
	go func() {
		for i := 1; i <= 5; i++ {
			ch <- i // 向通道写入数据
			time.Sleep(500 * time.Millisecond)
		}
		close(ch) // 关闭通道
	}()

	// 使用 for-range 读取数据
	for val := range ch {
		fmt.Println(val)
	}
	fmt.Println("Channel closed, for-range exited.")
}

```

# 4 管道发送和接收元素的本质是什么?

管道发送和接收元素的本质是什么？

> All transfer of value on the go channels happens with the copy of value.

就是说管道的发送和接收操作本质上都是 “值的拷贝”，无论是从 sender goroutine 的栈到 chan buf，还是从 
chan buf 到 receiver goroutine，或者是直接从 sender goroutine 到 receiver goroutine。

# 5 管道在什么情况下会引起资源泄漏？

Channel可能会引发 goroutine 泄漏。

泄漏的原因是 goroutine 操作 channel 后，处于发送或接收阻塞状态，而 channel 处于满或空的状态，一直得不到改变。同时，
垃圾回收器也不会回收此类资源，进而导致 goroutine 会一直处于等待队列中，不见天日。

另外，程序运行过程中，对于一个 channel，如果没有任何 goroutine 引用了，即便是它没有被关闭掉，gc 也会对其进行回收操作，
不会引起内存泄漏。

在构建超时返回机制时，我们应采用非阻塞型 channel

以下面这段典型的、借助阻塞型  channel  实现超时返回机制的函数代码为例。

```go
func handle(timeout time.Duration) *Obj {
    ch := make(chan *Obj)
    go func() {
        result := fn() // 逻辑处理
        ch <- result   // block
    }()
    select {
    case result := <-ch:
        return result
    case <-time.After(timeout):
        return nil
    }
}
```

当第 4 行在协程内执行的函数耗时较长，使得 handle 函数超时返回时，会导致阻塞型通道变量 ch 没有了接收者。这样一来，
第 5 行向通道写入数据的操作就会永远处于阻塞状态，最终引发协程泄漏问题。因此，为有效规避这一问题，在构建超时返回机制时，
我们应采用非阻塞型 channel，具体实现可参考后面的代码。

```go
func handle(timeout time.Duration) *Obj {
    //ch := make(chan *Obj)
    ch := make(chan *Obj, 1) // 使用非阻塞型channel
    go func() {
        result := fn() // 逻辑处理
        ch <- result   // block
    }()
    select {
    case result := <-ch:
        return result
    case <-time.After(timeout):
        return nil
    }
}
```


# 6 管道有哪些常用的应用场景?

## 6.1 停止信号

参见close-chan-safely，这块就略过了。
channel用于停止信号的场景还是挺多的，经常是关闭某个channel或者向channel发送一个元素，使得接收channel的那一方获知道
此信息，进而做一些其他的操作。

## 6.2 任务定时
与 timer 结合，一般有两种玩法：实现超时控制，实现定期执行某个任务。
有时候，需要执行某项操作，但又不想它耗费太长时间，上一个定时器就可以搞定：

```golang
select {
   case <- time.After(100 * time.Millisecond):
   case <- s.stopc:
      return false
}
```

```golang
package main

import (
	"fmt"
	"time"
)

func main() {
	ch := make(chan int)
	timer := time.NewTimer(3 * time.Second) // 设置 3 秒超时

	go func() {
		// 模拟长时间任务
		time.Sleep(5 * time.Second)
		ch <- 42 // 任务完成后向通道发送数据
	}()

	select {
	case val := <-ch:
		fmt.Println("Received:", val) // 如果任务在 3 秒内完成
	case <-timer.C:
		fmt.Println("Timeout!") // 如果 3 秒内未完成
	}
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

## 6.3 解耦生产者和消费者
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

## 6.4 控制并发数
有时需要定时执行几百个任务，例如每天定时按城市来执行一些离线计算的任务。但是并发数又不能太高，因为任务执行过程依赖第三方
的一些资源，对请求的速率有限制。这时就可以通过 channel 来控制并发数。

下面的例子来自《Go 语言高级编程》：

```golang
var limit = make(chan int, 3)

func main() {
    // …………
    for _, w := range work {
        go func(w work) {
            limit <- 1
            w()
            defer func() {
                <-limit
            }       
        }(w)
    }
    // …………
}
```

构建一个缓冲型的 channel，容量为 3。接着遍历任务列表，每个任务启动一个 goroutine 去完成。真正执行任务，访问第三方的动作在 w() 中完成，在执行 w() 之前，先要从 limit 中拿“许可证”，拿到许可证之后，才能执行 w()，并且在执行完任务，要将“许可证”归还。这样就可以控制同时运行的 goroutine 数。

这里，limit <- 1 放在 func 内部而不是外部，原因是：

**如果在外层，就是控制系统 goroutine 的数量，可能会阻塞 for 循环，影响业务逻辑**

**limit 其实和逻辑无关，只是性能调优，放在内层和外层的语义不太一样**

还有一点要注意的是，如果 w() 发生 panic，那"许可证"可能就还不回去了，因此需要使用 defer 来保证。

# 7 调度器交互：sudog、gopark 与 goready

前面提到 goroutine 在 channel 操作时会"阻塞"和"被唤醒"，但 runtime 层面实际发生了什么？

## 7.1 sudog 结构体

当一个 goroutine 因 channel 操作被阻塞时，runtime 会将它包装成一个 `sudog`（sudo-goroutine）结构体，挂到 channel 的等待队列中：

```go
// src/runtime/runtime2.go
type sudog struct {
    g       *g          // 被阻塞的 goroutine
    next    *sudog      // 等待队列中的下一个
    prev    *sudog      // 等待队列中的上一个
    elem    unsafe.Pointer // 指向发送/接收的数据
    c       *hchan      // 所在的 channel
    // ... 其他字段
}
```

`sudog` 是 goroutine 与 channel 之间的桥梁。一个 goroutine 可以同时出现在多个 channel 的等待队列中（例如 `select` 监听多个 channel 时），因此 `sudog` 和 `g` 是多对一的关系。

`sudog` 通过 P 本地缓存池（`sudogcache`）复用，避免频繁的堆分配。

## 7.2 gopark：挂起 goroutine

当 goroutine 需要等待 channel 时，runtime 调用 `gopark` 将当前 G 从 running 切换到 waiting 状态：

```
gopark 执行流程:
1. 将当前 G 的状态从 _Grunning → _Gwaiting
2. 解绑当前 G 与 M 的关系（dropg）
3. 调用 schedule() 让 M 去执行其他可运行的 G
4. 当前 G 停留在 waiting 状态，不参与调度
```

以向一个已满的 buffered channel 发送数据为例：

```go
// src/runtime/chan.go（简化）
func chansend(c *hchan, ep unsafe.Pointer, ...) bool {
    // ... 加锁，检查缓冲区已满 ...

    // 1. 获取 sudog
    gp := getg()
    mysg := acquireSudog()
    mysg.elem = ep      // 指向要发送的数据
    mysg.g = gp
    mysg.c = c
    c.sendq.enqueue(mysg) // 2. 入队

    // 3. 挂起当前 goroutine
    gopark(chanparkcommit, unsafe.Pointer(&c.lock), ...)

    // --- 被唤醒后从这里继续执行 ---
    releaseSudog(mysg)   // 4. 归还 sudog
    return true
}
```

**关键细节**：`gopark` 的第一个参数 `chanparkcommit` 是一个回调函数，在 G 状态切换后、释放 M 之前执行。对于 channel 操作，这个回调负责释放 `hchan.lock`，确保"解锁"和"挂起"是一个原子操作，避免竞态。

## 7.3 goready：唤醒 goroutine

当另一个 goroutine 从 channel 读取数据时（使得缓冲区不再满），runtime 调用 `goready` 唤醒等待中的 sender：

```
goready 执行流程:
1. 将目标 G 的状态从 _Gwaiting → _Grunnable
2. 将 G 放入当前 P 的本地运行队列（runq）的队首
3. 如果有空闲的 M，唤醒它来执行（wakep）
```

```go
// 简化的 chanrecv 中的唤醒逻辑
func chanrecv(c *hchan, ep unsafe.Pointer, ...) (bool, bool) {
    // ... 从缓冲区读取数据 ...

    // 如果 sendq 中有等待的 sender
    if sg := c.sendq.dequeue(); sg != nil {
        // 将 sender 的数据拷贝到缓冲区（或直接传递）
        // 唤醒 sender
        goready(sg.g, 3) // 第二个参数是 trace skip
    }
    // ...
}
```

## 7.4 直接传递优化

当无缓冲 channel 进行收发、或 buffered channel 缓冲区为空且有 receiver 等待时，runtime 会跳过缓冲区，将数据直接从 sender 的栈拷贝到 receiver 的栈：

```go
// src/runtime/chan.go（简化）
func send(c *hchan, sg *sudog, ep unsafe.Pointer, ...) {
    if sg.elem != nil {
        // 直接将数据从 sender 栈拷贝到 receiver 提供的地址
        sendDirect(c.elemtype, sg, ep)
        sg.elem = nil
    }
    goready(sg.g, 4)
}
```

这个优化减少了一次到缓冲区的内存拷贝，对于无缓冲 channel 尤其重要——数据直接从一个 goroutine 的栈复制到另一个 goroutine 的栈。

## 7.5 与 G/M/P 调度模型的关系

```
Channel 发送（缓冲区已满）:
  G1 (running on M1/P1)                G2 (waiting in sendq)
       |                                      |
  ch <- data                              <- 被 G1 之前某次 recv 唤醒
       |
  缓冲区已满，创建 sudog
       |
  gopark → G1 状态: running → waiting
       |
  M1 调用 schedule()
       |
  M1 从 P1.runq 取出 G3 继续执行

后续某个 goroutine 从 ch 读取数据:
  recv 操作发现 sendq 非空
       |
  goready(G1) → G1 状态: waiting → runnable
       |
  G1 被放入某个 P 的 runq
       |
  某个 M 拿到 G1 继续执行
```

重要特征：
- **gopark 不阻塞 M**：M 立即去执行其他 G，不会被浪费
- **goready 不立即执行**：只是将 G 放入 runq，等待被调度
- 这与操作系统线程阻塞不同——OS 线程阻塞会浪费整个线程

# 8 Go 内存模型与 channel 的 happens-before 保证

Go 内存模型（Go Memory Model）定义了在什么条件下，一个 goroutine 对变量的写入操作对另一个 goroutine 可见。channel 是最重要的同步原语之一。

## 8.1 happens-before 规则

Go 规范对 channel 定义了以下 happens-before 关系：

### 规则 1：send happens-before 对应的 receive 完成

> The kth send on a channel with capacity C happens before the (k+C)th receive from that channel completes.

```go
var msg string
ch := make(chan struct{}, 1) // C = 1

go func() {
    msg = "hello"        // (A)
    ch <- struct{}{}     // (B) 第 1 次 send
}()

<-ch                     // (C) 第 1 次 receive（k+C = 1+1 时的第 2 次不适用这里）
fmt.Println(msg)         // (D) 保证看到 "hello"
// 因为 (B) happens-before (C) 完成，且 (A) happens-before (B)
```

### 规则 2：close happens-before 收到零值

> The closing of a channel is synchronized before a receive that returns a zero value because the channel is closed.

```go
var data []int
ch := make(chan struct{})

go func() {
    data = []int{1, 2, 3}   // (A)
    close(ch)                 // (B) close
}()

<-ch                          // (C) 收到零值
fmt.Println(data)             // 保证看到 [1 2 3]
```

这是 `sync.Once` 和很多初始化模式的基础。

### 规则 3：无缓冲 channel 的 receive happens-before send 完成

> A receive from an unbuffered channel is synchronized before the send on that channel completes.

```go
var msg string
ch := make(chan struct{}) // 无缓冲

go func() {
    msg = "hello"
    ch <- struct{}{}   // (B) send，阻塞直到 receiver 就绪
}()

<-ch                   // (A) receive happens-before (B) 完成
fmt.Println(msg)       // 保证看到 "hello"
```

注意无缓冲 channel 和有缓冲 channel 的方向差异——无缓冲时 **receive 先于 send 完成**，这是因为 `chansend` 在发现有等待的 receiver 时会直接把数据拷贝到 receiver 的栈上，然后才返回。

## 8.2 不构成 happens-before 的操作

以下操作 **不提供** happens-before 保证：

```go
// len(ch) 不是同步操作！
if len(ch) > 0 {
    // 不能保证此时从 ch 读到的值和 len() 观测时一致
    // 也不能保证 sender 写入的共享状态对当前 goroutine 可见
}

// cap(ch) 同理，不是同步操作
```

这是一个常见的误区：开发者用 `len(ch) > 0` 来判断是否可以非阻塞读取，但这既有数据竞争的风险（其他 goroutine 可能同时在读），也不提供内存可见性保证。

> 陷阱演示 → [trap/happens-before](trap/happens-before/main.go)

## 8.3 实际意义

```go
// 模式1: 通过 channel 传递所有权（最安全）
ch := make(chan *Data)
go func() {
    d := &Data{Value: 42}   // 在 goroutine A 中构造
    ch <- d                   // 传递所有权给 goroutine B
    // 此后 goroutine A 不再访问 d
}()
d := <-ch
// 此后只有 goroutine B 访问 d，无需加锁

// 模式2: 通过 close 通知就绪（初始化模式）
ready := make(chan struct{})
go func() {
    initializeHeavyResource() // 初始化
    close(ready)               // 通知就绪
}()
<-ready
// 此后安全访问已初始化的资源
```

# 9 select 底层实现详解

前面提到 `select` 可以监控多个 channel，当多个可操作时随机选择。下面深入分析 `runtime.selectgo` 的实现细节。

## 9.1 selectgo 的执行流程

```go
// src/runtime/select.go
func selectgo(cas0 *scase, order0 *uint16, ...) (int, bool)
```

`selectgo` 的完整执行分为 4 个阶段：

```
阶段 1: 生成随机排列（pollorder）和锁排序（lockorder）
    ↓
阶段 2: 按 pollorder 遍历所有 case，检查是否有立即可执行的
    ↓ 如果有 → 执行并返回
    ↓ 如果没有 →
阶段 3: 将当前 goroutine 加入所有 case 对应 channel 的等待队列
    ↓
    gopark 挂起
    ↓ （被某个 channel 操作唤醒）
阶段 4: 从所有等待队列中移除，返回被唤醒的 case
```

## 9.2 随机排列（pollorder）

为了保证公平性，`selectgo` 不按源码顺序遍历 case，而是生成一个随机排列：

```go
// 使用 cheaprandn（低开销伪随机）生成 Fisher-Yates shuffle
norder := 0
for i := range ncases {
    j := cheaprandn(uint32(norder + 1))
    pollorder[norder] = pollorder[j]
    pollorder[j] = uint16(i)
    norder++
}
```

这就是"多个 channel 同时就绪时随机选择"的实现方式。如果不随机化，总是优先检查第一个 case，会导致后面的 case 被饿死。

## 9.3 锁排序（lockorder）—— 防止死锁

当需要阻塞等待时（阶段 3），`selectgo` 必须同时操作多个 channel 的等待队列。为了避免死锁，它按照 `hchan` 的内存地址对 channel 排序，然后按顺序加锁：

```go
// 按 hchan 地址排序
func sellock(scases []scase, lockorder []uint16) {
    var c *hchan
    for _, o := range lockorder {
        c0 := scases[o].c
        if c0 != c { // 跳过同一个 channel（去重）
            c = c0
            lock(&c.lock)
        }
    }
}
```

**为什么按地址排序？** 这是经典的死锁预防策略——所有 goroutine 按同一个全局顺序获取锁，就不会形成环形等待。

例如，两个 goroutine 同时 select 同样的 channel A 和 B：
- 如果不排序：G1 锁 A 再锁 B，G2 锁 B 再锁 A → 死锁
- 按地址排序：两者都先锁地址较小的 → 不会死锁

## 9.4 default 分支的编译优化

```go
select {
case v := <-ch:
    // ...
default:
    // ...
}
```

带 `default` 的 select 在编译时不会走 `selectgo`，而是被优化为调用 `selectnbrecv`（非阻塞接收）：

```go
// 编译器生成的伪代码
if selectnbrecv(&v, ch) {
    // case 分支
} else {
    // default 分支
}
```

`selectnbrecv` 内部调用 `chanrecv(c, elem, false)`，第三个参数 `block=false` 表示不阻塞。这比完整的 `selectgo` 高效得多。

## 9.5 reflect.Select：动态数量的 channel

标准的 `select` 语句中 case 数量必须在编译期确定。如果需要在运行时动态监听不确定数量的 channel，可以使用 `reflect.Select`：

```go
import "reflect"

cases := make([]reflect.SelectCase, len(channels))
for i, ch := range channels {
    cases[i] = reflect.SelectCase{
        Dir:  reflect.SelectRecv,
        Chan: reflect.ValueOf(ch),
    }
}

// 动态 select
chosen, value, ok := reflect.Select(cases)
fmt.Printf("case %d: received %v (ok=%v)\n", chosen, value, ok)
```

**注意**：`reflect.Select` 有显著的性能开销（反射、内存分配），仅在真正需要动态 channel 数量时使用。静态场景永远优先用编译期 `select`。

## 9.6 nil channel 在 select 中的妙用

nil channel 的 case 在 select 中会被永久跳过，这个特性可以用来动态禁用某个 case：

```go
// 合并两个 channel，某个关闭后禁用它
for ch1 != nil || ch2 != nil {
    select {
    case v, ok := <-ch1:
        if !ok { ch1 = nil; continue } // 关闭后设为 nil
        process(v)
    case v, ok := <-ch2:
        if !ok { ch2 = nil; continue }
        process(v)
    }
}
```

这是实现 fan-in（多路合并）模式的基础技巧。

> 陷阱演示 → [trap/select-nil-chan](trap/select-nil-chan/main.go)

# 10 makechan 内存分配策略

调用 `make(chan T, size)` 时，runtime 的 `makechan` 函数根据元素类型和缓冲区大小采用不同的分配策略：

```go
// src/runtime/chan.go（简化）
func makechan(t *chantype, size int) *hchan {
    elem := t.Elem

    // 情况 1: 无缓冲 channel
    if size == 0 || elem.Size_ == 0 {
        c = (*hchan)(mallocgc(hchanSize, nil, true))
        c.buf = c.raceaddr()
    }

    // 情况 2: 元素不含指针
    else if !elem.Pointers() {
        // hchan 和 buf 一次性分配（连续内存）
        c = (*hchan)(mallocgc(hchanSize+elem.Size_*uintptr(size), nil, true))
        c.buf = add(unsafe.Pointer(c), hchanSize)
    }

    // 情况 3: 元素含指针
    else {
        // hchan 和 buf 分开分配
        c = new(hchan)
        c.buf = mallocgc(elem.Size_*uintptr(size), elem, true)
    }
    // ...
}
```

### 三种情况的对比

| 条件 | 分配方式 | 原因 |
|------|---------|------|
| 无缓冲 / 元素大小为 0 | 只分配 hchan | 不需要缓冲区 |
| 元素不含指针 (如 `chan int`) | hchan + buf 连续分配 | GC 不需要扫描 buf 中的指针 |
| 元素含指针 (如 `chan *User`) | hchan 和 buf 分开分配 | GC 需要单独扫描 buf 中的指针 |

**为什么元素含指针时要分开分配？**

Go 的 GC 是精确 GC（precise GC），需要知道内存块中哪些位置存放了指针。如果 hchan 和 buf 连续分配，且 buf 中存储的是指针类型，GC 扫描时需要用元素类型的信息来遍历 buf 区域。分开分配后，buf 的类型信息独立，GC 可以直接按元素类型扫描。

### 内存布局示意

```
元素不含指针（chan int, bufSize=4）:
┌──────────────────────────────────────────┐
│  hchan (96 bytes)  │  buf (4 × 8 bytes)  │
└──────────────────────────────────────────┘
      一次 mallocgc 调用，连续内存

元素含指针（chan *User, bufSize=4）:
┌──────────────────┐     ┌─────────────────────┐
│  hchan (96 bytes) │     │  buf (4 × 8 bytes)  │
└──────────────────┘     └─────────────────────┘
  new(hchan)              mallocgc with elem type
  独立分配                 独立分配，带类型信息
```

# 11 Channel vs Mutex vs Atomic 性能对比

"不要通过共享内存来通信，而应该通过通信来共享内存"——但这并不意味着所有场景都应该使用 channel。不同的同步原语有不同的性能特征和适用场景。

## 11.1 三种同步原语的本质开销

| 原语 | 操作开销 | 涉及的 runtime 操作 |
|------|---------|-------------------|
| Channel | 高 | hchan.lock 加锁、缓冲区拷贝、可能的 gopark/goready |
| Mutex | 中 | 快速路径：atomic CAS；慢路径：信号量 + 调度 |
| Atomic | 低 | CPU 指令级别（LOCK CMPXCHG 等） |

### Pingpong 延迟

两个 goroutine 来回传递消息，测量单次往返延迟：

- **Channel**: 每次往返 = 2 次加锁 + 2 次数据拷贝 + 可能的 goroutine 切换
- **Mutex+Cond**: 每次往返 = 2 次 Lock/Unlock + 2 次 Signal/Wait

Channel 通常比 Mutex+Cond 慢 2-3 倍，因为 channel 操作路径更长。

### 扇入计数器

N 个 goroutine 竞争递增同一个计数器：

- **Atomic**: 直接在 CPU 缓存行上 CAS，无需系统调用
- **Mutex**: 快速路径仅需一次 CAS，但竞争时退化为内核调度
- **Channel**: 每次递增都要经过 channel 发送 + 接收，开销最大

在高竞争的简单计数场景下，性能排序：**Atomic >> Mutex >> Channel**

## 11.2 选型指南

```
需要在 goroutine 间传递数据的所有权？
├── 是 → Channel（"不要通过共享内存来通信"）
└── 否 → 保护共享状态
         ├── 简单的标志位/计数器？ → Atomic
         ├── 需要保护一段临界区代码？ → Mutex/RWMutex
         └── 需要条件等待？ → Mutex + Cond（或 Channel）
```

经验法则：
- **数据流动**（producer→consumer、pipeline、fan-in/fan-out）→ Channel
- **状态保护**（缓存、配置、连接池）→ Mutex
- **性能计数器**（metrics、stats）→ Atomic
- **一次性通知**（ready、done、cancel）→ Channel（或 `context.Context`）

> 基准测试 → [performance/chan_vs_mutex_test.go](performance/chan_vs_mutex_test.go)

# 12 Channel 缓冲区大小选型

缓冲区大小的选择直接影响程序的吞吐量、内存占用和语义正确性。

## 12.1 无缓冲 vs 有缓冲

| 特性 | 无缓冲 (size=0) | 有缓冲 (size>0) |
|------|----------------|----------------|
| 同步语义 | 强同步：send 和 receive 必须同时就绪 | 弱同步：send 只要缓冲区未满即可返回 |
| 延迟 | 每次 send 都可能阻塞 | 缓冲区未满时 send 不阻塞 |
| 吞吐量 | 低（收发强耦合） | 高（收发解耦） |
| 内存 | 最小 | 缓冲区大小 × 元素大小 |
| 数据竞争检测 | 更容易发现 | 可能掩盖时序问题 |

## 12.2 缓冲区大小与吞吐量的关系

基准测试显示，吞吐量随缓冲区大小呈对数增长：

```
buf=0:    基准值（每次 send 都阻塞等待 receive）
buf=1:    显著提升（解耦了收发的瞬时速度差）
buf=10:   持续提升
buf=100:  提升幅度递减
buf=1000: 几乎无额外收益，但内存开销增大
```

> 基准测试 → [performance/chan_buffer_size_test.go](performance/chan_buffer_size_test.go)

## 12.3 选型原则

**1. 已知确切数量 → 使用该数量**

```go
// 有 N 个 goroutine 各产生一个结果
results := make(chan Result, N)
for range N {
    go func() { results <- compute() }()
}
// 收集所有结果，channel 永远不会阻塞 send
```

**2. 信号通知 → 容量为 0 或 1**

```go
done := make(chan struct{})    // 0: 同步通知
quit := make(chan struct{}, 1) // 1: 防止 send 阻塞（如超时场景）
```

**3. 生产者-消费者流水线 → 适度缓冲**

```go
// 缓冲区 = 预期的突发量（burst size）
// 通常 64-256 是一个好的起点
pipeline := make(chan Task, 128)
```

**4. 不要用大缓冲区掩盖消费速度不足**

```go
// 错误：消费者处理不过来，用 10 万的缓冲区"解决"
ch := make(chan Event, 100000)

// 正确：增加消费者数量或优化消费速度
ch := make(chan Event, 128)
for range runtime.NumCPU() {
    go consumer(ch)
}
```

**5. 超时返回场景 → 容量为 1**

这个在第 5 节已经讨论过：

```go
ch := make(chan *Obj, 1) // 容量为 1，防止超时后 goroutine 泄漏
go func() {
    ch <- fn()  // 即使无人接收，也不会永久阻塞
}()
select {
case result := <-ch:
    return result
case <-time.After(timeout):
    return nil
}
```