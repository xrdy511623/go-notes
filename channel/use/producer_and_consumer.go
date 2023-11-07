package main

import (
	"fmt"
	"sync"
	"time"
)

var chan1 = make(chan int, 1000000)
var chan3 = make(chan bool, 1000)
var res = make([]int, 0)
var mutex = sync.Mutex{}

/*
任务:找出1-1000000的自然数中的素数
*/

func producer(srcChan chan int) {
	//defer wg.Done()
	for i := 1; i <= 1000000; i++ {
		srcChan <- i
	}
	// 发送数据完成，关闭通道srcChan
	close(srcChan)
}

func consumer(srcChan chan int, signalChan chan bool) {
	for v := range srcChan {
		flag := true
		for i := 2; i <= v/2; i++ {
			if v%i == 0 {
				flag = false
				break
			}
		}
		if flag {
			mutex.Lock()
			res = append(res, v)
			mutex.Unlock()
		}
	}
	signalChan <- true

}

func main() {
	start := time.Now()
	go producer(chan1)
	for i := 1; i <= 1000; i++ {
		// 同时开启大量的消费者协程去同一个管道chan1消费数据，由于管道中的数据
		// 是消费(取出)一个少一个，取数据又是只读操作，所以不会有资源竞争的问题
		// 这样多个goroutine并发的消费数据，便可以大大提高工作效率。
		go consumer(chan1, chan3)
	}

	/*
		不使用sync的goroutine同步机制，也可以通过程序设计实现主线程等到所有的协程执行完毕后再退出。
		这里主线程对chan2进行循环取值操作，只要chan2通道不关闭，这个循环就不会停止，这样主程序便不会退出，
		主线程下开启的生产者和多个消费者协程便可以自在的执行了。那么主程序何时退出呢？我们知道对通道channel
		进行for range循环取值操作，如果该通道channel被关闭，那么该循环便会退出，此时主程序也就退出了。
		不过问题的关键在于，开启大量的goroutine对同一个通道chan2写入数据，我们应该在什么时候关闭chan2通道以确保
		主线程的for range chan2顺利退出呢？这里引入了一个用于判断通道chan3，每开启一个消费者协程，一旦消费者任务里
		对通道chan2写入数据的操作完成，便会向判断通道chan3发送一个信号true,这样当通道chan3的数据取完时，就意味着
		所有的消费者协程对通道chan2写入数据完成，此时便可以关闭通道chan2了。
	*/
	for i := 0; i < 1000; i++ {
		<-chan3
	}
	close(chan3)
	// 一共有78499个素数
	//fmt.Println(len(chan2))
	fmt.Println(len(res))
	end := time.Now()
	costTime := end.Sub(start)
	/*
		case1
		如果线性执行，处理十万数据，耗时2.04秒
		开启10个协程，处理十万数据，耗时219.23ms
		开启100个协程，处理十万数据，耗时161.95ms
		开启1000个协程，处理十万数据，耗时163ms
		case2
		开启10个协程，处理一百万数据，耗时16.57s
		开启100个协程，处理一百万数据，耗时13.19s
		开启1000个协程，处理一百万数据，耗时12.96秒
	*/
	fmt.Printf("共花费%v时间\n", costTime)

}
