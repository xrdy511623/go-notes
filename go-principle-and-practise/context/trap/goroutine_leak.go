package main

import (
	"context"
	"fmt"
	"time"
)

func genBad() <-chan int {
	ch := make(chan int)
	go func() {
		var n int
		for {
			ch <- n
			n++
			time.Sleep(time.Second)
		}
	}()
	return ch
}

func genGood(ctx context.Context) <-chan int {
	ch := make(chan int)
	go func() {
		var n int
		for {
			select {
			case <-ctx.Done():
				return
			case ch <- n:
				n++
				time.Sleep(time.Second)
			}
		}
	}()
	return ch
}

func main() {
	// 下面这种使用方式会造成genBad中的协程泄露
	//for n := range genBad() {
	//	fmt.Println(n)
	//	if n == 5 {
	//		break
	//	}
	//}
	// ……

	// 解决方案就是使用context来做并发控制，等到满足条件时发出取消信号
	ctx, cancel := context.WithCancel(context.Background())
	// 避免其他地方忘记 cancel，且重复调用不影响
	defer cancel()
	for n := range genGood(ctx) {
		fmt.Println(n)
		if n == 5 {
			cancel()
			break
		}
	}
	// ……
}
