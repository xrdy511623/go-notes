package main

import (
	"fmt"
	"time"
)

var chan1 = make(chan int, 1000000)
var chan2 = make(chan int, 1000000)
var chan3 = make(chan struct{}, 10)
var res = make([]int, 0)

/*
任务:找出1-1000000的自然数中的素数
*/

// check 校验一个自然数是否是素数
func check(num int) bool {
	flag := true
	for i := 2; i <= num/2; i++ {
		if num%i == 0 {
			flag = false
			break
		}
	}
	return flag
}

func plain() {
	for i := 1; i <= 1000000; i++ {
		if check(i) {
			res = append(res, i)
		}
	}
}

func producer(srcChan chan int) {
	for i := 1; i <= 1000000; i++ {
		srcChan <- i
	}
	// 发送数据完成，关闭通道srcChan
	close(srcChan)
}

func consumer(srcChan chan int, resChan chan int, signalChan chan struct{}) {
	for v := range srcChan {
		// 多个协程向同一个通道发数据，并不会出现并发问题。
		// 因为channel底层有lock用来保证每个读channel或写channel的操作都是原子的。
		if check(v) {
			resChan <- v
		}
	}
	signalChan <- struct{}{}
}

func main() {
	t1 := time.Now()
	plain()
	// 一共有78499个素数
	fmt.Println(len(res))
	fmt.Printf("线性执行共花费%v时间\n", time.Since(t1))

	t2 := time.Now()
	go producer(chan1)
	for i := 1; i <= 10; i++ {
		go consumer(chan1, chan2, chan3)
	}
	/*
		不使用sync的goroutine同步机制(waitGroup)，也可以通过程序设计实现主线程等到所有的协程执行完毕后再退出。
		这里主协程对chan3进行循环取值操作，只要chan3通道还有数据，这个循环就不会停止，这样主程序便不会退出，
		主线程下开启的生产者和多个消费者协程便可以自在地执行了。那么主程序何时退出呢？这里的关键在于，开启多个
		goroutine向同一个通道chan2写入数据，我们应该在什么时候关闭chan2通道？这里引入了一个用于判断的通道chan3，
		每开启一个消费者协程，一旦消费者任务里对通道chan2写入数据的操作完成，便会向判断通道chan3发送一个信号空结构体,
		这样当通道chan3的数据取完时，就意味着所有的消费者协程对通道chan2写入数据完成，此时便可以关闭通道chan2了。
	*/
	for i := 0; i < 10; i++ {
		<-chan3
	}
	close(chan2)
	// chan3可以关闭，也可以不关闭，不关闭的话GC会回收，不会造成泄露
	// close(chan3)
	// 一共有78499个素数
	fmt.Println(len(chan2))
	/*
		线性执行共花费14.19s
		开启10个协程，处理一百万数据，耗时2.42s
	*/
	fmt.Printf("共花费%v时间\n", time.Since(t2))
}
