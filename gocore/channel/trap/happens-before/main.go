package main

import (
	"fmt"
	"sync"
)

/*
陷阱：channel 的 happens-before 保证与常见误用

运行：go run .

Go 内存模型对 channel 定义了明确的 happens-before 规则：
  1. 有缓冲 channel: 第 k 次 send happens-before 第 k+C 次 receive 完成（C = capacity）
  2. 无缓冲 channel: receive happens-before send 完成（注意方向！）
  3. close happens-before 从 closed channel 接收到零值
  4. 容量为 0 的 channel 的第 k 次 receive happens-before 第 k 次 send 完成

本示例演示：
  - 正确利用 happens-before 保证跨 goroutine 的数据可见性
  - len(ch) 不构成同步点，不能用来判断是否有数据可安全读取
*/

func main() {
	fmt.Println("=== 演示1: 无缓冲 channel 的同步保证 ===")
	demoUnbufferedHappensBefore()

	fmt.Println("\n=== 演示2: close 的 happens-before 保证 ===")
	demoCloseHappensBefore()

	fmt.Println("\n=== 演示3: len(ch) 不是同步点（陷阱） ===")
	demoLenNotSyncPoint()

	fmt.Println("\n=== 演示4: 有缓冲 channel 可做信号量的 happens-before 保证 ===")
	demoBufferedSemaphore()
}

// demoUnbufferedHappensBefore 无缓冲 channel：receive 先于 send 完成
// 这意味着 send 返回时，receiver 一定已经拿到了值
func demoUnbufferedHappensBefore() {
	ch := make(chan struct{})
	msg := ""

	go func() {
		msg = "hello from goroutine" // (1) 写入 msg
		ch <- struct{}{}             // (2) send，会阻塞直到 main 接收
	}()

	<-ch // (3) receive happens-before (2) 完成
	// 因此 (1) happens-before (3)，msg 对 main goroutine 可见
	fmt.Printf("msg = %q（保证可见）\n", msg)
}

// demoCloseHappensBefore close happens-before 收到零值
func demoCloseHappensBefore() {
	ch := make(chan struct{})
	data := make([]int, 0, 3)

	go func() {
		data = append(data, 1, 2, 3) // (1) 写入 data
		close(ch)                    // (2) close
	}()

	<-ch // (3) 收到零值，且 (2) happens-before (3)
	// 因此 (1) happens-before (3)，data 对 main goroutine 可见
	fmt.Printf("data = %v（保证可见）\n", data)
}

// demoLenNotSyncPoint len(ch) 不提供任何同步保证
// 即使 len(ch) > 0，也不能保证此刻读到的数据是最新的
func demoLenNotSyncPoint() {
	ch := make(chan int, 1)
	sharedState := 0

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		sharedState = 42 // (1) 写入共享状态
		ch <- 1          // (2) 向 channel 发送信号
	}()

	wg.Wait()

	// 错误用法：用 len() 判断是否有数据
	// len(ch) 不构成 happens-before 关系，存在理论上的数据竞争风险
	if len(ch) > 0 {
		fmt.Printf("len(ch) = %d，但 len() 不是同步操作！\n", len(ch))
		fmt.Println("虽然这里因为 wg.Wait() 已经同步了，但如果只依赖 len(ch)，")
		fmt.Println("Go 内存模型不保证 sharedState=42 对当前 goroutine 可见。")
	}

	// 正确用法：通过 receive 操作建立 happens-before
	v := <-ch
	fmt.Printf("通过 <-ch 接收到 %d，此时 sharedState=%d 保证可见\n", v, sharedState)
}

// demoBufferedSemaphore 有缓冲 channel 的 happens-before：
// 第 k 次 send happens-before 第 k+C 次 receive 完成
// 容量为 C 的 channel 可以限制并发数，同时提供内存可见性保证
func demoBufferedSemaphore() {
	const limit = 2
	sem := make(chan struct{}, limit)
	results := make([]string, 4)

	var wg sync.WaitGroup
	for i := range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{} // 获取信号量（第 k 次 send）
			results[i] = fmt.Sprintf("task-%d done", i)
			<-sem // 释放信号量（第 k+C 次 receive 之前的某次 receive）
		}()
	}
	wg.Wait()
	fmt.Printf("results = %v\n", results)
}
