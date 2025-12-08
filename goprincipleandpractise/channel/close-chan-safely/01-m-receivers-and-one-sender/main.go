package main

import (
	"fmt"
	"sync"
	"time"
)

/*
如何优雅地关闭通道？
关闭一个 nil channel会导致panic，所以使用nil channel时需要格外小心。
由于关闭一个已经关闭的通道会引发panic,所以在不知道一个通道是否已经关闭的时候关闭此通道是很危险的。
向一个已关闭的通道发送数据也会引发panic，所以在不知道一个通道是否已经关闭的时候向此通道发送数据是很危险的。
那么我们应该如何优雅地关闭一个通道呢？
通道关闭原则
一个常用地使用Go通道的原则是不要在数据接收方或者在有多个发送者的情况下关闭通道，换句话说，
我们只应该让一个通道唯一的发送者关闭此通道。
我们将称此原则为通道关闭原则。
*/

/*
情形一：M个接收者和一个发送者。发送者通过关闭用来传输数据的通道来传递发送结束信号
这是最简单的一种情形。当发送者欲结束发送，让它关闭用来传输数据的通道即可。
*/

func check(num int) bool {
	for i := 2; i <= num/2; i++ {
		if num%i == 0 {
			return false
		}
	}
	return true
}

func main() {
	// 判断1-1000000之间有多少个素数，并输出所有素数。素数指的是这个数只能被1和它自己整除，譬如5就是一个素数
	const Max = 1000000
	const NumReceivers = 1000
	wg := &sync.WaitGroup{}
	wg.Add(NumReceivers)
	dataChan := make(chan int, Max)
	resChan := make(chan int, Max)
	start := time.Now()
	// 发送者
	go func() {
		for i := 1; i < Max; i++ {
			dataChan <- i
		}
		// 发送完毕后关闭传输数据的通道dataChan
		close(dataChan)
	}()

	// 接收者
	for j := 0; j < NumReceivers; j++ {
		go func() {
			defer wg.Done()
			for num := range dataChan {
				if check(num) {
					resChan <- num
				} else {
					continue
				}
			}
		}()
	}
	wg.Wait()
	length := len(resChan)
	ret := make([]int, length)
	for i := 0; i < length; i++ {
		v := <-resChan
		ret[i] = v
	}
	close(resChan)
	fmt.Println(len(ret))
	fmt.Printf("cost time:%v\n", time.Since(start))
}
