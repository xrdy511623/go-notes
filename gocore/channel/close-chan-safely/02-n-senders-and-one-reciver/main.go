package main

import (
	"fmt"
	"math/rand"
	"sync"
)

/*
情形二：一个接收者和N个发送者，此唯一接收者通过关闭一个额外的信号通道来通知发送者不要再发送数据了
对于此额外的信号通道stopCh，它只有一个发送者，即dataCh数据通道的唯一接收者。 dataCh数据通道的接收者
关闭了信号通道stopCh，这是不违反通道关闭原则的。
在下面的案例中，数据通道dataCh并没有被关闭。是的，我们不必关闭它。 当一个通道不再被任何协程所使用后，
它将逐渐被垃圾回收掉，无论它是否已经被关闭。 所以这里的优雅性体现在通过不关闭一个通道来停止使用此通道。
*/

func main() {
	const Max = 10000
	const NumSenders = 10
	dataChan := make(chan int)
	stopChan := make(chan struct{})
	wg := &sync.WaitGroup{}
	wg.Add(1)
	for i := 0; i < NumSenders; i++ {
		go func() {
			for {
				// The try-receive operation is to try
				// to exit the goroutine as early as
				// possible. For this specified example,
				// it is not essential.
				select {
				case <-stopChan:
					return
				default:
				}
				select {
				case <-stopChan:
					return
				case dataChan <- rand.Intn(Max):
				}
			}
		}()
	}

	go func() {
		defer wg.Done()
		for value := range dataChan {
			if value == Max-1 {
				close(stopChan)
				fmt.Println(value)
				return
			}
		}
	}()
	wg.Wait()
}
