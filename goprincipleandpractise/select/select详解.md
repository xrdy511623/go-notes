
---
select详解
---

select是Go在语言层面提供的I/O多路复用机制，用于检测多个管道是否就绪(即可读或可写)，其特性跟管道息息相关。

# 1 select的特性

## 1.1 管道读写

select只能作用于管道，包括数据的读取和写入，如以下代码所示:

```go
func SelectForChan(c chan string) {
	var recv string
	sned := "Hello"
	
	select {
	case recv = <- c:
		fmt.Printf("received %s\n", recv)
    case c <- send:
		fmt.Printf("sent %s\n", send)
    }
}
```
在上面的代码中，select拥有两个case语句，分别对应管道的读操作和写操作，至于最终执行哪个case语句，取决于函数传入的管道。

### 1.1.1 第一种情况，管道没有缓冲区

如以下代码所示:

```go
c := make(chan string)
SelectForChan(c)
```

此时函数传入的是一个无缓冲区的通道，必须有一个协程向它写入数据，另一个协程从它那读取数据，否则此时管道既不能读也不能写，所以
两个case语句均不执行，select陷入阻塞。

### 1.1.2 第一种情况，管道有缓冲区且还可以存放至少一个数据

如以下代码所示:

```go
c := make(chan string, 1)
SelectForChan(c)
```

此时管道可以写入，写操作对应的case语句得到执行，且执行结束后函数退出。

### 1.1.3 第三种情况，管道有缓冲区且缓冲区已放满数据

如以下代码所示:

```go
c := make(chan string, 1)
c <- "Hello"
SelectForChan(c)
```

此时管道可以读取，读操作对应的case语句得到执行，且执行结束后函数退出。

### 1.1.4 第四种情况，管道有缓冲区，缓冲区中已有部分数据且还可以存入数据

如以下代码所示:

```go
c := make(chan string, 2)
c <- "Hello"
SelectForChan(c)
```

此时管道既可以读取也可以写入，select将随机挑选一个case语句执行，任意一个case语句执行结束后函数就退出。


## 1.2 小结

综上所述，select的每个case语句只能操作一个管道，要么写入数据，要么读取数据。鉴于管道的特性，如果管道中没有数据，
读取操作则会阻塞；如果管道中没有空余的缓冲区(缓冲区已满)则写入操作会阻塞；当select的多个case语句中的管道均阻塞时，
整个select语句也会陷入阻塞(没有default语句的情况下)，直到任意一个管道解除阻塞。如果多个case语句均没有阻塞，那么
select将随机挑选一个case语句执行。


## 1.3 返回值

select为Go语言的预留关键字，并非函数，其可以在case语句中声明变量并为变量赋值，看上去就像一个函数。

case语句读取管道时，可以最多给两个变量赋值，如以下代码所示:

```go
func selectAssign(c chan string) {
	select {
	// 0个变量
	case <- c:
		fmt.Printf("0\n")
    // 1个变量
    case d := <- c:
	    fmt.Printf("1: received %s\n", d)
    // 两个变量
    case d, ok := <- c:
		if !ok {
            fmt.Printf("no data found\n")
        }
        fmt.Printf("2: received %s\n", d)
    }
}
```

case语句中管道的读操作有两个返回条件，一是成功读到数据，二是管道中已没有数据且已被关闭。当case语句中包含两个变量时，第二个
变量表示时候成功地读出了数据。

下面的代码传入一个关闭的通道:

```go
c := make(chan string)
close(c)
selectAssign(c)
```

此时select中的三个case语句都有机会执行，第二个和第三个case语句收到的数据都为空，但第三个case语句可以感知到管道被关闭，从而
不必打印空数据。

## 1.4 default
select中的default语句不能处理管道读写操作，当select的所有case语句都阻塞时，default语句将被执行，如以下代码所示:

```go
func SelectDefault() {
	c := make(chan string)
	select {
	case <- c:
	    fmt.Printf("received\n")
    default:
	    fmt.Printf("no data found in default\n")
    }
}
```

上面的管道由于没有缓冲区，读操作必然阻塞，然而select含有default分支，select将执行default分支并退出。

另外，default实际上是特殊的case，它能出现在select中的任意位置，但每个select仅能出现一次。


# 2 select 使用场景

## 2.1 永久阻塞
有时我们启动协程处理任务，并且不希望main函数退出，此时就可以让main函数永久性陷入阻塞。

在Kubernetes项目的多个组件中均有使用select阻塞main函数的案例，比如apiserver中的webhook测试组件：

```go
func main() {
	server := webhooktesting.NewTestServer(nil)
	server.StartTLS()
	fmt.Println("serving on", server.URL)
	select {}
}
```

以上代码的select语句中不包含case语句和default语句，那么协程(main)将陷入永久性的阻塞。

## 2.2 快速检错

有时我们会使用管道来传输错误。此时就可以使用select语句来快速检查管道中是否有错误并且避免陷入循环。

比如Kubernetes调度器中就有类似的用法:

```go
errChan := make(chan error, active)
// 传入管道用于记录错误
jm.deleteJobPods(&job, activePods, errCh)

select {
// 检查是否有错误发生
case manageJobErr = <- errChan:
	if manageJobErr != nil {
		break
    }
default:
	// 没有错误，快速结束检查。
}
```

上面的select仅用于尝试从管道中读取错误信息，如果没有错误，则不会陷入阻塞。


## 2.3 限时等待

有时我们会使用管道来管理函数的上下文，此时可以使用select来创建只有一定时效的管道。
比如Kubernetes控制器中就有类似的用法:

```go
func waitForStopOrTimeout(stopChan <-chan struct{}, timeout time.Duration) <- chan struct {} {
	stopChWithTimeout := make(chan struct{})
	go func() {
		select {
		case <- stopCh:
			// 自然结束
        case <- time.After(timeout):
			// 最长等待时长
        }
		close(stopChWithTimeout)
    }
	return stopChWithTimeout
}
```

该函数返回一个管道，可用于在函数之间传递，但该管道会在指定时间后自动关闭。