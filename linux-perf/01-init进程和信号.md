
---
init进程和信号
---

# 1 如何理解init进程

Linux内核执行文件一般会放在/boot目录下，文件名类似vmlinuz*。在内核完成了操作系统的各种初始化之后，这个程序需要执行的第一个
用户态进程就是init进程。
系统启动的时候先是执行内核态的代码，然后在内核中调用1号进程的代码，从内核态切换到用户态。
目前主流的Linux发行版，都会把/sbin/init作为符号链接指向Systemd。Systemd是目前最流行的Linux Init进程，在它之前还有
SysVinit、UpStart等Linux Init进程。

**init进程的基本功能**

无论是哪种Linux init进程，它最基本的功能都是创建出Linux系统中其他所有的进程，并且管理这些进程。

在Linux上有了容器的概念之后，一旦容器创建了自己的PID Namespace（进程命名空间），这个Namespace里的进程号也是从1开始标记的。
所以，容器的init进程也被称为1号进程。也就是说，1号进程是第一个用户态的进程，由它直接或者间接创建了Namespace中的其他进程。

## 1.1 PID 1 的两大特殊职责

PID 1 不仅仅是"第一个进程"这么简单，它在内核层面有两项其他进程不具备的特殊职责。

**职责一：僵尸进程收割（Zombie Reaping）**

当一个子进程退出时，内核会保留它的退出状态信息（进程描述符），直到父进程调用`wait()`/`waitpid()`读取。在这个等待期间，
退出的子进程处于Z（Zombie）状态，在`ps`输出中显示为`<defunct>`。

正常情况下，父进程会负责`wait()`自己的子进程。但如果父进程先于子进程退出，这些"孤儿进程"会被reparent到PID 1。
此时PID 1必须调用`wait()`来回收它们，否则僵尸进程会持续积累。

```
进程退出流程：

  子进程退出 ──> 变成Zombie ──> 父进程wait() ──> 内核回收进程描述符
                    │
                    │ 如果父进程已退出
                    v
              reparent到PID 1 ──> PID 1 wait() ──> 内核回收
                    │
                    │ 如果PID 1不做wait()
                    v
              Zombie持续积累 ──> PID耗尽 ──> 系统无法创建新进程
```

可以用以下命令查看系统中的僵尸进程数量：

```bash
# 查看僵尸进程数
ps aux | awk '{if($8=="Z") print}' | wc -l

# 或直接看 /proc 统计
cat /proc/loadavg   # 第4列 "running/total" 中total包含zombie
```

**职责二：信号转发（Signal Forwarding）**

当系统关闭或容器停止时，PID 1会收到SIGTERM信号。一个合格的init进程应该将这个信号转发给它的所有子进程，
让它们有机会进行优雅退出（graceful shutdown）——保存状态、关闭连接、刷新缓冲区等。

如果PID 1不转发信号，子进程在SIGTERM超时后（docker stop默认10秒）会被SIGKILL强制杀死，
没有任何清理的机会，可能导致数据丢失或状态不一致。

# 2 如何理解Linux信号

用一句话来概括，信号其实就是Linux进程收到的一个**异步通知**。

Linux信号分为两类：标准信号（1~31）和实时信号（34~64）。日常工作中主要接触标准信号。

下面是几个典型的触发场景：
- 通过键盘按下`Ctrl+C`，当前运行的进程会收到SIGINT（2号信号）而退出
- 代码写得有问题，导致内存访问出错，当前进程会收到SIGSEGV（11号信号）
- 通过命令`kill <pid>`发送信号，缺省不指定信号类型时发送的是SIGTERM（15号信号）
- 指定信号类型`kill -9 <pid>`，发送的是SIGKILL（9号信号），进程无法拒绝

## 2.1 常用信号速查表

| 信号 | 编号 | 默认行为 | 可捕获 | 典型触发场景 |
|------|------|---------|--------|-------------|
| SIGHUP | 1 | 终止 | 是 | 终端断开；守护进程惯例用于重载配置（如nginx -s reload） |
| SIGINT | 2 | 终止 | 是 | 键盘Ctrl+C |
| SIGQUIT | 3 | 终止+core dump | 是 | 键盘Ctrl+\\；Go程序收到后会打印所有goroutine的栈 |
| SIGKILL | 9 | 终止 | **否** | kill -9；内核直接终止进程，不执行任何用户代码 |
| SIGBUS | 7 | 终止+core dump | 是 | 非法内存地址对齐访问 |
| SIGSEGV | 11 | 终止+core dump | 是 | 非法内存访问（空指针、越界等） |
| SIGPIPE | 13 | 终止 | 是 | 写入已关闭的pipe或socket；Go默认忽略此信号 |
| SIGTERM | 15 | 终止 | 是 | kill的默认信号；docker stop首先发送此信号 |
| SIGCHLD | 17 | 忽略 | 是 | 子进程退出时内核发送给父进程 |
| SIGCONT | 18 | 继续执行 | 是 | 恢复被SIGSTOP暂停的进程（fg命令） |
| SIGSTOP | 19 | 暂停 | **否** | 内核暂停进程，不可捕获（Ctrl+Z发送的是SIGTSTP，可捕获） |
| SIGURG | 23 | 忽略 | 是 | socket收到Out-of-Band数据 |

> 完整列表可通过`kill -l`查看，或`man 7 signal`查看每个信号的详细说明。

## 2.2 进程处理信号的三个选择

进程在收到信号后，有三个选择：

**忽略（Ignore）** —— 对信号不做任何处理。但SIGKILL和SIGSTOP是例外，进程不能忽略这两个信号，
因为它们是Linux内核和超级用户用于强制控制任意进程的特权信号。

**捕获（Catch）** —— 进程注册自己的handler来处理信号。一旦注册了handler，收到信号时内核不再执行默认行为，
而是调用用户注册的handler函数。

**缺省行为（Default）** —— 使用Linux内核为每个信号定义的默认行为。大部分信号的默认行为是终止进程。
对于大部分信号，应用程序使用系统缺省行为就可以了。

## 2.3 特权信号与容器中的init进程

SIGKILL（9号）和SIGSTOP（19号）是两个特权信号，不能被捕获也不能被忽略。

在容器中，init进程（PID 1）对信号的处理有特殊规则：Linux内核会把只有default handler的信号都忽略掉
（仅针对PID Namespace内的init进程）。这意味着：

- SIGKILL和SIGSTOP因为不能注册handler，只有default handler，所以对容器内PID 1无效
- SIGTERM等信号如果PID 1没有注册handler，同样会被忽略
- 只有PID 1显式注册了handler的信号才会被响应

这就是为什么直接把业务进程当作容器PID 1时，`docker stop`可能无法优雅终止进程——
如果业务进程没有注册SIGTERM的handler，内核会替PID 1忽略掉这个信号，
最后只能等10秒超时后被SIGKILL强杀。

## 2.4 信号的系统调用：signal() vs sigaction()

**发送信号：kill()**

`kill()`系统调用有两个参数：`sig`（信号编号，如15表示SIGTERM）和`pid`（目标进程号）。
在shell中对应的就是`kill`命令。

```bash
kill 1234       # 发送SIGTERM（默认）
kill -9 1234    # 发送SIGKILL
kill -SIGUSR1 1234  # 发送SIGUSR1
```

**注册handler：signal() vs sigaction()**

`signal()`是早期UNIX的接口，存在两个问题：
1. 在某些系统上，handler执行一次后会被重置为default，需要在handler内重新注册（不可靠）
2. 无法获取信号的附加信息（如发送者PID）
3. 行为在不同UNIX实现上不一致

`sigaction()`是POSIX标准接口，解决了上述所有问题，生产环境推荐使用。

```c
// signal() —— 简单但不推荐用于生产
signal(SIGTERM, my_handler);

// sigaction() —— 可靠，推荐使用
struct sigaction sa;
sa.sa_handler = my_handler;
sigemptyset(&sa.sa_mask);
sa.sa_flags = SA_RESTART;  // 被信号中断的系统调用自动重启
sigaction(SIGTERM, &sa, NULL);
```

如果只是想忽略一个信号，可以将handler设为SIG_IGN：

```c
signal(SIGPIPE, SIG_IGN);  // 忽略SIGPIPE，网络编程中非常常见
```

## 2.5 信号投递机制与处理时机

信号是异步的，但进程并不是在收到信号的瞬间就执行handler。实际的处理时机是：

```
发送者调用kill() ──> 内核设置目标进程的pending信号位 ──> ... ──> 目标进程从内核态返回用户态时
                                                                   内核检查pending信号
                                                                   ──> 调用handler
```

也就是说，信号的处理发生在**进程从内核态返回用户态的时刻**（系统调用返回、中断返回、上下文切换恢复执行时）。
如果进程长时间处于不可中断睡眠状态（D状态），信号会一直pending，直到进程醒来。

可以通过`/proc/<pid>/status`查看进程的信号状态：

```bash
$ cat /proc/1/status | grep Sig
SigQ:   0/63377          # 当前排队信号数 / 上限
SigPnd: 0000000000000000 # 线程级pending信号（位图，16进制）
ShdPnd: 0000000000000000 # 进程级pending信号
SigBlk: 0000000000000000 # 被阻塞的信号（信号掩码）
SigIgn: 0000000000001000 # 被忽略的信号（第13位=SIGPIPE）
SigCgt: 0000000180004002 # 被捕获的信号（已注册handler）
```

上面的16进制位图可以转成二进制来确定具体是哪些信号。例如SigIgn的`1000`（16进制）= 第13位置1 = SIGPIPE被忽略。

# 3 信号处理代码示例

## 3.1 Go：优雅处理SIGTERM实现Graceful Shutdown

这是Go服务端程序最常见的信号处理模式：

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
)

func main() {
    srv := &http.Server{Addr: ":8080"}

    // 启动HTTP服务
    go func() {
        if err := srv.ListenAndServe(); err != http.ErrServerClosed {
            log.Fatalf("HTTP server error: %v", err)
        }
    }()

    // 等待SIGTERM或SIGINT
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
    sig := <-quit
    log.Printf("received signal: %v, shutting down...", sig)

    // 给在途请求最多5秒完成
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := srv.Shutdown(ctx); err != nil {
        log.Fatalf("server forced to shutdown: %v", err)
    }
    log.Println("server exited gracefully")
}
```

要点：
- `signal.Notify`注册感兴趣的信号，Go runtime会把信号投递到channel
- `srv.Shutdown(ctx)`会停止接收新连接，等待在途请求完成（带超时）
- Go默认忽略SIGPIPE，捕获SIGURG（runtime内部使用），SIGQUIT会触发goroutine dump

## 3.2 Shell：作为容器init进程的信号转发脚本

当不方便修改业务进程代码时，可以用shell脚本做信号转发：

```bash
#!/bin/bash
# entrypoint.sh —— 容器入口，负责信号转发

# 启动业务进程（后台运行）
/app/server &
CHILD_PID=$!

# 捕获SIGTERM和SIGINT，转发给子进程
trap "echo 'Forwarding SIGTERM to $CHILD_PID'; kill -TERM $CHILD_PID" SIGTERM SIGINT

# 等待子进程退出
wait $CHILD_PID
EXIT_CODE=$?

echo "Child exited with code $EXIT_CODE"
exit $EXIT_CODE
```

# 4 容器场景深入：为什么需要tini

## 4.1 直接把业务进程当PID 1的问题

很多Dockerfile这样写：

```dockerfile
CMD ["./myapp"]
```

此时myapp直接成为容器的PID 1，存在两个问题：

1. **僵尸进程泄漏**：myapp fork出的子进程如果变成孤儿再退出，没人做`wait()`，zombie堆积
2. **信号处理缺失**：如果myapp没有注册SIGTERM handler，`docker stop`发送的SIGTERM会被内核忽略（PID 1特殊规则），
   只能等超时后被SIGKILL强杀

## 4.2 解决方案

**方案一：docker run --init**

Docker内置了一个轻量级init进程（基于tini），添加`--init`即可：

```bash
docker run --init myimage
```

或在docker-compose.yml中：

```yaml
services:
  myapp:
    image: myimage
    init: true    # 启用tini作为PID 1
```

此时容器内的进程树变为：

```
PID 1: /sbin/docker-init (tini)
  └── PID 2: ./myapp
        └── PID 3: worker-thread
```

tini负责：收割zombie + 将SIGTERM转发给myapp。

**方案二：在Dockerfile中集成tini**

```dockerfile
RUN apt-get update && apt-get install -y tini
ENTRYPOINT ["tini", "--"]
CMD ["./myapp"]
```

**方案三：Kubernetes的处理**

Kubernetes 1.28+ 支持SidecarContainers特性，但对PID 1问题，通常做法是：
- 设置`shareProcessNamespace: true`让Pod内容器共享PID Namespace
- 或在镜像中集成tini/dumb-init
- `terminationGracePeriodSeconds`控制SIGKILL前的等待时间（默认30秒）

```yaml
spec:
  terminationGracePeriodSeconds: 30
  containers:
  - name: myapp
    image: myimage
    # 确保业务代码处理SIGTERM
```

# 5 性能视角

作为linux-perf系列的第一篇，有必要从性能角度审视init进程和信号。

## 5.1 僵尸进程对系统性能的影响

僵尸进程本身不占用CPU和内存（代码段和数据段已释放），但它仍占用：
- 一个进程描述符（task_struct，约数KB）
- 一个PID编号

当zombie大量积累时：
- PID耗尽（默认上限`/proc/sys/kernel/pid_max`，通常为32768或4194304），系统无法`fork()`新进程
- `/proc`目录膨胀，`ps`等命令变慢
- 监控系统报告大量进程，干扰问题排查

```bash
# 快速检查zombie数量
ps aux | awk '$8 ~ /Z/ {count++} END {print "Zombie count:", count+0}'

# 找到zombie的父进程，定位问题根因
ps -eo pid,ppid,stat,cmd | awk '$3 ~ /Z/'
```

## 5.2 信号处理与上下文切换的关系

信号处理发生在内核态→用户态切换时，这与CPU上下文切换（本系列第03篇）直接相关：

1. **系统调用返回时**：内核在返回用户态之前检查pending信号，如有则先跳转到handler
2. **handler执行本身就是一次额外的用户态代码执行**：如果handler中包含系统调用，会产生额外的内核态切换
3. **频繁的信号会增加上下文切换次数**：例如大量子进程同时退出时，父进程会收到大量SIGCHLD

## 5.3 SIGSTOP/SIGCONT对CPU统计的影响

`SIGSTOP`会将进程置于T（Stopped）状态，此时进程不占用CPU时间片。但需要注意：
- 进程持有的锁不会释放，可能导致其他进程阻塞
- 如果被stop的进程是关键服务（如数据库），会引发连锁超时
- `top`和`pidstat`中stopped进程的CPU使用率会降为0，但其关联进程的等待时间会增加

# 6 实用命令速查

```bash
# 查看所有信号列表及编号
kill -l

# 查看进程树（含PID）
pstree -p

# 查看某进程的信号掩码状态
cat /proc/<pid>/status | grep -E 'Sig(Q|Pnd|Blk|Ign|Cgt)|ShdPnd'

# 追踪进程收到和发送的信号（strace）
strace -e trace=signal -p <pid>

# 查看系统中的僵尸进程
ps aux | awk '$8 ~ /Z/'

# 查看容器内PID 1是什么
docker exec <container> cat /proc/1/cmdline | tr '\0' ' '

# 向进程组发送信号（负数PID表示进程组）
kill -TERM -<pgid>

# 查看PID上限
cat /proc/sys/kernel/pid_max
```
