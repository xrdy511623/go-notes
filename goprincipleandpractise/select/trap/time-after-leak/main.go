package main

import (
	"fmt"
	"runtime"
	"time"
)

/*
陷阱：for-select 中使用 time.After 导致 Timer 泄漏

运行：go run .

预期行为：
  每次循环调用 time.After 都会创建一个新的 time.Timer，且该 Timer
  在触发前不会被 GC 回收（被 runtime timer 堆引用）。
  如果循环频繁且 timeout 较长，大量 Timer 积压会导致内存持续增长。

  正确做法：使用 time.NewTimer + Reset 复用单个 Timer。
*/

func main() {
	fmt.Println("=== 演示: for-select 中 time.After 的 Timer 泄漏 ===")

	ch := make(chan int, 1)

	// 启动一个缓慢的生产者
	go func() {
		for i := range 2000 {
			ch <- i
			time.Sleep(50 * time.Microsecond) // 每 50us 发一个
		}
		close(ch)
	}()

	// ---------- 错误做法：每轮循环创建新 Timer ----------
	fmt.Println("\n--- 错误做法: time.After 每次创建新 Timer ---")
	count := 0
	var msBefore runtime.MemStats
	runtime.ReadMemStats(&msBefore)

	for {
		select {
		case v, ok := <-ch:
			if !ok {
				goto done
			}
			_ = v
			count++
		case <-time.After(1 * time.Second):
			// 这里的 time.After 每次循环都会创建新的 Timer
			// 即使 ch 在 1 秒内就有数据，旧 Timer 仍然存在于 runtime timer 堆中
			fmt.Println("timeout")
			goto done
		}
	}
done:
	runtime.GC()
	var msAfter runtime.MemStats
	runtime.ReadMemStats(&msAfter)
	fmt.Printf("  处理了 %d 条消息\n", count)
	fmt.Printf("  堆内存: %d KB → %d KB（Timer 尚未触发时内存更高）\n",
		msBefore.HeapInuse/1024, msAfter.HeapInuse/1024)

	// ---------- 正确做法：复用 Timer ----------
	fmt.Println("\n--- 正确做法: time.NewTimer + Reset 复用单个 Timer ---")

	ch2 := make(chan int, 1)
	go func() {
		for i := range 2000 {
			ch2 <- i
			time.Sleep(50 * time.Microsecond)
		}
		close(ch2)
	}()

	timer := time.NewTimer(1 * time.Second)
	defer timer.Stop() // 确保退出时释放

	count = 0
	runtime.ReadMemStats(&msBefore)

	for {
		select {
		case v, ok := <-ch2:
			if !ok {
				goto done2
			}
			_ = v
			count++
			// 收到数据后重置 Timer，始终只有一个 Timer 存在
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(1 * time.Second)
		case <-timer.C:
			fmt.Println("timeout")
			goto done2
		}
	}
done2:
	runtime.GC()
	runtime.ReadMemStats(&msAfter)
	fmt.Printf("  处理了 %d 条消息\n", count)
	fmt.Printf("  堆内存: %d KB → %d KB（始终只有 1 个 Timer）\n",
		msBefore.HeapInuse/1024, msAfter.HeapInuse/1024)

	fmt.Println("\n总结:")
	fmt.Println("  1. time.After 在 for-select 中会每轮创建新 Timer，造成 GC 压力")
	fmt.Println("  2. 正确做法: time.NewTimer 创建一次，循环中用 Stop+Reset 复用")
	fmt.Println("  3. Reset 前必须先 Stop 并排空 channel，否则可能收到旧的超时信号")
}
