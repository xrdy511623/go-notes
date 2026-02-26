package main

import (
	"fmt"
	"runtime"
	"time"
)

/*
陷阱：在循环中使用 time.After 导致内存泄漏

运行：go run .

预期行为：
  time.After(d) 等价于 time.NewTimer(d).C。
  每次调用都创建一个新的 Timer，但返回的是 channel 而非 Timer 本身，
  因此调用者无法 Stop 它。在 Go 1.23 之前，未到期的 Timer 不会被 GC 回收。

  在 select 循环中，如果消息频率高于超时时长，每次迭代创建的 Timer
  在到期前都不会释放，造成内存持续增长。

  正确做法：使用 time.NewTimer 并手动 Reset
*/

func main() {
	fmt.Println("=== 错误做法：select 循环中使用 time.After ===")

	msgChan := make(chan int, 1)

	// 快速发送消息，使 time.After 的 Timer 来不及到期
	go func() {
		for i := 0; i < 100000; i++ {
			msgChan <- i
		}
		close(msgChan)
	}()

	var memBefore, memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	count := 0
	for {
		select {
		case _, ok := <-msgChan:
			if !ok {
				goto done
			}
			count++
		case <-time.After(time.Minute): // 每次迭代创建新 Timer！
			fmt.Println("timeout")
		}
	}
done:

	runtime.GC()
	runtime.ReadMemStats(&memAfter)
	fmt.Printf("  处理 %d 条消息\n", count)
	fmt.Printf("  总分配字节: %d KB\n", (memAfter.TotalAlloc-memBefore.TotalAlloc)/1024)
	fmt.Printf("  总分配对象: %d\n", memAfter.Mallocs-memBefore.Mallocs)

	fmt.Println("\n=== 正确做法：复用 Timer ===")

	msgChan2 := make(chan int, 1)
	go func() {
		for i := 0; i < 100000; i++ {
			msgChan2 <- i
		}
		close(msgChan2)
	}()

	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	timer := time.NewTimer(time.Minute)
	defer timer.Stop()

	count = 0
	for {
		select {
		case _, ok := <-msgChan2:
			if !ok {
				goto done2
			}
			count++
			// 复用同一个 Timer
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(time.Minute)
		case <-timer.C:
			fmt.Println("timeout")
		}
	}
done2:

	runtime.GC()
	runtime.ReadMemStats(&memAfter)
	fmt.Printf("  处理 %d 条消息\n", count)
	fmt.Printf("  总分配字节: %d KB\n", (memAfter.TotalAlloc-memBefore.TotalAlloc)/1024)
	fmt.Printf("  总分配对象: %d\n", memAfter.Mallocs-memBefore.Mallocs)

	fmt.Println("\n总结:")
	fmt.Println("  1. time.After 每次调用都创建新 Timer，无法 Stop")
	fmt.Println("  2. 在高频 select 循环中会造成大量 Timer 堆积")
	fmt.Println("  3. 正确做法：使用 time.NewTimer + Reset 复用")
	fmt.Println("  4. Go 1.23+ 未引用的 Timer 可被 GC，但分配开销仍在")
}
