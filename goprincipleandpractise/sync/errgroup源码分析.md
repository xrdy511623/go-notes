
---
errgroup 源码分析(扩展)
---

# 1 WaitGroup

## 1.1 简介

倘若我们需要向多个下游发起并发请求，并且必须得等待所有请求返回时，常规解决方法是用 Golang 的基础并发类型 WaitGroup。
WaitGroup 的作用是阻塞等待多个并发任务执行完成。WaitGroup 类型主要包含下面几个方法。


```go
func (wg *WaitGroup) Add(delta int)
func (wg *WaitGroup) Done()
func (wg *WaitGroup) Wait()
```

第一个是 Add 方法，在任务运行之前，需要调用 Add 方法，用于设置需要等待完成的任务数，Add 方法传进去的数值之和，需要和任务数相等。
第二个是 Done 方法，每个任务完成时，需要调用 Done 方法，用于告知 WaitGroup 对象已经有一个任务运行完成。
第三个是 Wait 方法，当需要等待所有并发任务完成时，调用 Wait 方法，用于阻塞主协程。


下面是使用WaitGroup的例子。

```go
import (
    "sync"
)

var urls = []string{
    "http://www.golang.org/",
    "http://www.google.com/",
    "http://www.somestupidname.com/",
}

func TestWaitGroup(t *testing.T) {
    // 创建WaitGroup对象
    wg := sync.WaitGroup{}
    results := make(chan string, len(urls))
    for index, url := range urls {
        url := url
        index := index
        // 在创建协程执行任务之前，调用Add方法
        wg.Add(1)
        go func() {
            // 任务完成后，调用Done方法
            defer wg.Done()
            // Fetch the URL.
            resp, err := http.Get(url)
            if err != nil {
                return
            }

            defer resp.Body.Close()
            body, err := io.ReadAll(resp.Body)
            if err != nil {
                return
            }
            results <- string(body)

        }()
    }
    // 主协程阻塞，等待所有的任务执行完成
    wg.Wait()
}
```

## 1.2 局限性

虽然使用 WaitGroup 类型可以实现并发等待功能，但是 WaitGroup 类型在错误处理方面存在一定的功能局限性。
比如， 当某个并发任务产生错误时，我们难以便捷地将该错误信息传递至主协程进行处理，而且无法有效地中止其他待运行或运行中的任务。
这一点其实非常重要，因为它可以避免因继续执行这些不必要的任务而导致的资源浪费。
另外，WaitGroup也不支持控制并发运行的最大协程数。我们知道，go的协程虽然是轻量级的线程，开销不大，但在并发任务数量很大的情况下
(比如达到数十万甚至百万级别)，如果每一个并发任务都开一个协程，一方面，协程的切换调度开销会成为瓶颈，另一方面，系统的内存消耗也会随之激增，
可能会导致系统资源的消耗过多，甚至导致系统崩溃。


# 2 errgroup 的使用

为了方便我们对并发任务的错误进行处理，Golang 为我们提供了极为便捷地并发扩展库 ——errgroup 包。errgroup 包的核心是
Group 类型，Group 类型是对 WaitGroup 的封装，在并发等待的基础功能之上，它额外提供了一系列实用的扩展功能。

## 2.1 错误处理：如何在主协程中获取并发任务错误信息？

首先咱们来看看错误处理功能，也就是当并发任务执行出错的时候，主协程能获取到出错信息并进行处理。这就涉及到 Group 类型
提供的两个重要方法。
首先是 Go 方法，该方法的参数为具有错误返回值的函数类型，在 Go 方法内部会在一个独立的协程中去
运行所传入的这个函数，这样就能实现并发执行的效果。
其次便是 Wait 方法，它与 WaitGroup 类型的 Wait 方法有点像， Group 类型的 Wait 方法同样会阻塞等待所有传入到
Go 方法中的函数全部运行完毕。然而，二者的不同之处也十分明显，Group 类型的 Wait 方法具备一个 error 类型的返回值。
如果传入 Go 方法的函数运行返回了错误，那么此 Wait 方法将会返回该错误信息。当然，在多个传入的函数中都出现了错误时，
它只会返回所遇到的第一个错误。


```go
func (g *Group) Go(f func() error)
func (g *Group) Wait() error
```

在了解了 Group 类型的 API 之后，我们再看看如何使用 errgroup 包。

用 errgroup 包实现并发等待和错误处理功能的示例代码如下。

```go
import (
    "golang.org/x/sync/errgroup"
)

func TestErrHandle(t *testing.T) {
    results := make(chan string, len(urls))
    // 创建Group类型
    g := new(errgroup.Group)
    for index, url := range urls {
        // Launch a goroutine to fetch the URL.
        url := url
        index := index
        // 调用Go方法
        g.Go(func() error {
            // Fetch the URL.
            resp, err := http.Get(url)
            if err != nil {
                return err // 返回错误
            }
            defer resp.Body.Close()
            body, err := io.ReadAll(resp.Body)
            if err != nil {
                return err // 返回错误
            }
            results <- string(body)
            return nil
        })
    }
    // Wait for all HTTP fetches to complete.
    // 等待所有任务执行完成，并对错误进行处理
    if err := g.Wait(); err != nil {
        fmt.Println("Failed to fetched all URLs.")
    }
}
```

这段代码也很好理解，核心是下面几步。
第一步，我们要创建 Group 类型的对象。
第二步，在 Group 的 Go 方法中传入那些需要并发运行的函数。特别需要注意的是，这些传入的函数必须将错误返回。
第三步，也是最后一步，在主协程中，我们需要调用 Group 对象的 Wait 方法。通过这一调用，主协程将会阻塞等待，直至
所有通过 Go 方法传入的任务都执行完毕。并且，在任务完成后，我们还能够对 Wait 方法所返回的错误进行处理。


## 2.2 任务取消：如何中止并发任务的运行？

除了错误处理功能，errgroup 包提供的任务取消功能也相当实用。所谓任务取消功能，是指一旦在多个并发任务中有一个任务执行失败，
我们能够中止那些尚未开始执行或者是正在执行过程中的其他并发任务。这样一来，就可以避免因继续执行这些不必要的任务而导致资源浪费。
而要实现任务取消功能，除了我们之前学到的 Group 类型的 Go 方法和 Wait 方法，还有一个极为关键的核心方法，那就是 WithContext 函数。
借助这个函数，我们能够基于 context 来创建 Group 对象，这样在任务执行报错时，我们就可以利用 context 来停止所有相关任务。


```go
func WithContext(ctx context.Context) (*Group, context.Context)
```

在了解了 WithContext 函数之后，现在让我们 errgroup 包来实现任务取消功能。如同下面的代码，和前面实现错误处理功能不同，
用 errgroup 包来实现任务取消功能，有两个核心要点。第一点，需要用 WithContext 函数创建 Group 对象。
第二点，在传入 Go 方法的函数中，需要实现 select-done 模式，也就是当函数运行时，发现 context 被取消，则直接返回，
从而避免执行业务逻辑。


```go

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "time"
    
    "golang.org/x/sync/errgroup"
)

var urls = []string{ "http://www.golang.org/", "http://www.google.com/", "http://www.somestupidname.com/"}


func TestCancel(t *testing.T) {
    results := make(chan string, len(urls))
    timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
    // 用WithContext函数创建Group对象
    eg, ctx := errgroup.WithContext(timeoutCtx)
    for _, url := range urls {
        url := url
        // 调用Go方法
        eg.Go(func() error {
            req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
            if err != nil {
                return fmt.Errorf("create request for %s: %w", url, err)
            }

           resp, err := http.DefaultClient.Do(req)
		   if err != nil {
                return fmt.Errorf("fetch %s: %w", url, err)
            }
           defer resp.Body.Close()

           body, err := io.ReadAll(resp.Body)
           if err != nil {
                return fmt.Errorf("read body from %s: %w", url, err)
		   }   

            // 安全地写入结果（带超时检查）
            select {
            case results <- string(body):
                t.Logf("Success: %s (%d bytes)", url, len(body))
               return nil
            case <- ctx.Done():
                return ctx.Err()
            }
        })
    }
    // Wait for all HTTP fetches to complete.
    // 等待所有任务执行完成，并对错误进行处理
    if err := eg.Wait(); err != nil {
        fmt.Println("Failed to fetched all URLs.")
    }
	close(results)
}
```

## 2.3 协程数限制：如何控制并发运行的最大协程数？

除了错误处理和任务取消功能，errgroup 包还可以限制同时并发运行的最大协程数。它的核心是 SetLimit 方法。
如果我们用 SetLimit 方法设置了可同时运行的最大协程数，当调用 Go 方法时，一旦达到了最大协程数，就会阻塞创建新协程运行任务，
直到有协程运行完，才可以创建新协程。

```go
func (g *Group) SetLimit(n int)
```

现在让我们基于 SetLimit 方法，来实现限制并发运行的最大协程数功能。如同下面的代码一样，限制并发运行的最大协程数，
核心是调用 SetLimit 方法。

```go
import (
    "context"
    "fmt"
    "io"
    "net/http"
    "time"
    
    "golang.org/x/sync/errgroup"
)

var urls = []string{ "http://www.golang.org/", "http://www.google.com/", "http://www.somestupidname.com/"}


func TestCancel(t *testing.T) {
    results := make(chan string, len(urls))
    timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
    // 用WithContext函数创建Group对象
    eg, ctx := errgroup.WithContext(timeoutCtx)
    // 调用SetLimit方法，设置可同时运行的最大协程数 
    eg.SetLimit(2)
    for _, url := range urls {
        url := url
        // 调用Go方法
        eg.Go(func() error {
            req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
            if err != nil {
                return fmt.Errorf("create request for %s: %w", url, err)
            }

           resp, err := http.DefaultClient.Do(req)
		   if err != nil {
                return fmt.Errorf("fetch %s: %w", url, err)
            }
           defer resp.Body.Close()

           body, err := io.ReadAll(resp.Body)
           if err != nil {
                return fmt.Errorf("read body from %s: %w", url, err)
		   }   

            // 安全地写入结果（带超时检查）
            select {
            case results <- string(body):
                t.Logf("Success: %s (%d bytes)", url, len(body))
               return nil
            case <- ctx.Done():
                return ctx.Err()
            }
        })
    }
    // Wait for all HTTP fetches to complete.
    // 等待所有任务执行完成，并对错误进行处理
    if err := eg.Wait(); err != nil {
        fmt.Println("Failed to fetched all URLs.")
    }
	close(results)
}
```

# 3 errgroup 是如何实现的？

首先，咱们来看看 errgroup 包 Group 类型的数据结构，它有几个重要的成员变量。

```go
type token struct{}

type Group struct {
    cancel func(error) // 这个作用是為了WithContext 而來的

    wg sync.WaitGroup // errGroup底层的阻塞等待功能，就是通过WaitGroup实现的

    sem chan token // 用于控制最大运行的协程数

    err     error // 最后在Wait方法中返回的error
    errOnce sync.Once // 用于安全的设置err，只设置一次err
}
```

先说 cancel 函数，它就是为了前面讲的 WithContext 功能设计的，专门用来取消任务运行。
sync.WaitGroup 类型的变量 wg，Group 类型在底层是靠封装它来实现阻塞等待功能的。
通道类型变量 sem ，用来控制同时能跑的最大协程数量。
error 类型的变量 err，在 Group 类型的 Wait 方法里，它会被返回给调用者。
最后就是 sync.Once 类型变量，它的用处就是能安全地设置 err 变量。在多个协程一起并发跑的情况下，
它能保证 err 变量只设置一次，而且是并发安全的。

接着，咱们来看看 WithContext 函数和 SetLimit 方法，它们分别是任务取消功能和协程数限制的核心方法。
就像下面的代码一样，WithContext 函数内部会生成可以被取消的 context 类型对象 ctx，并且会生成 cancel 函数变量，
封装在 Group 对象中。SetLimit 方法则用于设置通道容量。


```go
func WithContext(ctx context.Context) (*Group, context.Context) {
    ctx, cancel := withCancelCause(ctx)
    return &Group{cancel: cancel}, ctx // 生成有取消功能的context
}

func (g *Group) SetLimit(n int) {
    g.sem = make(chan token, n) // 设置通道容量
}
```

然后，咱们来看看 Go 方法，这个方法是 Group 类型的核心方法。

```go
func (g *Group) Go(f func() error) {
    if g.sem != nil {
        g.sem <- token{} // 通道满则阻塞，用来控制最大并发协程数
    }

    g.wg.Add(1)
    go func() {
        defer g.done() // 底层调用g.wg.Done()

        if err := f(); err != nil {
            g.errOnce.Do(func() { // 安全的设置err变量
                g.err = err
                if g.cancel != nil {
                    g.cancel(g.err) // 任务运行出错，调用g.cancel方法，用context控制其它任务是否中止运行
                }
            })
        }
    }()
}

func (g *Group) done() {
    if g.sem != nil {
        <-g.sem // 协程运行完，通道读
    }
    g.wg.Done()
}
```

下面来分析下这个方法的实现。
假如我们调用 SetLimit 方法设置了通道容量，当需要创建协程之前，在 Go 方法的第 3 行会往通道写， 当通道已经被写满时则
阻塞不创建协程。当协程运行完时，在 Go 方法第 8 行的 done 方法内，会读通道。
第 6 行和第 8 行， 实际上就是调用 WaitGroup 类型的 Add 和 Done 方法来实现并发等待功能。
第 10-12 行，当传入 Go 方法的函数调用出错时，通过 errOnce 安全地设置 err 变量，这个错误便是 Wait 方法收到的错误。
第 14 行，调用 g.cancel 函数来让 ctx.Done() 能读到值，达到中止其它任务运行的目的。

最后，让我们来看看 Wait 方法。

```go
// Wait blocks until all function calls from the Go method have returned, then
// returns the first non-nil error (if any) from them.
func (g *Group) Wait() error {
    g.wg.Wait()
    if g.cancel != nil {
            g.cancel(g.err)
    }
    return g.err
}
```

Wait 方法比较简单，核心就是调用 g.wg 来阻塞等待，并将 Go 方法中设置的 g.err 错误返回，从而实现错误处理功能。