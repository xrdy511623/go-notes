package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync"
)

/*
情形三：M个接收者和N个发送者。它们中的任何协程都可以让一个中间调解协程帮忙发出停止数据传送的信号
这是最复杂的一种情形。我们不能让接收者和发送者中的任何一个去关闭用来传输数据的通道，我们也不能让多个接收者
之一关闭一个额外的信号通道。 这两种做法都违反了通道关闭原则。 然而，我们可以引入一个中间调解者角色并让其
关闭额外的信号通道来通知所有的接收者和发送者结束工作。 具体实现见下例。注意其中使用了一个尝试发送操作来
向中间调解者发送信号。

在下面的案例中，通道关闭原则依旧得到了遵守。
请注意，信号通道sigChan的容量必须至少为1。 如果它的容量为0，则在中间调解者还未准备好的情况下就已经有某个
协程向sigChan发送信号时，此信号将被抛弃。
*/

func main() {
	const Max = 100000
	const NumReceivers = 10
	const NumSenders = 100
	wg := &sync.WaitGroup{}
	wg.Add(NumReceivers)
	dataChan := make(chan int)
	stopChan := make(chan struct{})
	sigChan := make(chan string, 1)
	var sig string
	// 中间调解者
	go func() {
		sig = <-sigChan
		close(stopChan)
	}()

	// 发送者
	for i := 0; i < NumSenders; i++ {
		go func(id string) {
			for {
				value := rand.Intn(Max)
				// Here, the try-send operation is
				// to notify the moderator to close
				// the additional signal channel.
				if value == 0 {
					select {
					case sigChan <- "发送者" + id:
					default:
					}
					return
				}
				// The try-receive operation here is to
				// try to exit the sender goroutine as
				// early as possible. Try-receive and
				// try-send select blocks are specially
				// optimized by the standard Go
				// compiler, so they are very efficient.
				select {
				case <-stopChan:
					return
				default:
				}
				// Even if stopCh is closed, the first
				// branch in this select block might be
				// still not selected for some loops
				// (and for ever in theory) if the send
				// to dataCh is also non-blocking. If
				// this is unacceptable, then the above
				// try-receive operation is essential.
				select {
				case <-stopChan:
					return
				case dataChan <- value:
				}
			}
		}(strconv.Itoa(i))
	}

	// 接收者
	for i := 0; i < NumReceivers; i++ {
		go func(id string) {
			defer wg.Done()
			for {
				// Same as the sender goroutine, the
				// try-receive operation here is to
				// try to exit the receiver goroutine
				// as early as possible.
				select {
				case <-stopChan:
					return
				default:
				}
				// Even if stopCh is closed, the first
				// branch in this select block might be
				// still not selected for some loops
				// (and forever in theory) if the receive
				// from dataCh is also non-blocking. If
				// this is not acceptable, then the above
				// try-receive operation is essential.
				select {
				case <-stopChan:
					return
				case value := <-dataChan:
					if value == Max-1 {
						fmt.Println(value)
						// Here, the same trick is
						// used to notify the moderator
						// to close the additional
						// signal channel.
						select {
						case sigChan <- "接收者" + id:
						default:
						}
						return
					}
				}
			}
		}(strconv.Itoa(i))
	}
	wg.Wait()
	fmt.Println("被" + sig + "终止了")
}
