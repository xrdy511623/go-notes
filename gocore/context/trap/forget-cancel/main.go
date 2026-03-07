package main

import (
	"context"
	"fmt"
	"runtime"
	"time"
)

/*
陷阱：忘记调用 cancel() 导致资源泄漏

运行：go run .

预期行为：
  WithTimeout / WithDeadline 内部创建了 time.Timer。
  如果不调用 cancel()，即使 context 超时后，timerCtx 仍会保留在父节点的 children map 中，
  直到父节点被取消或 GC 回收。在长生命周期的父 context 下，这会导致内存持续增长。

  正确做法：始终 defer cancel()，即使你预期 context 会自然超时。
*/

func main() {
	fmt.Println("=== 演示: 忘记 cancel() 导致的资源泄漏 ===")

	// 使用 Background 作为长生命周期的父 context
	parentCtx := context.Background()

	// 模拟循环创建 WithTimeout 但不调用 cancel
	fmt.Println("\n--- 错误做法: 不调用 cancel ---")
	for i := range 1000 {
		//nolint:lostcancel // 故意不调用 cancel 来演示泄漏
		ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
		// 忘记调用 cancel()！
		// timer 会在 10 秒后触发，在此之前一直占用资源
		_ = ctx
		_ = cancel
		if i%200 == 0 {
			var ms runtime.MemStats
			runtime.ReadMemStats(&ms)
			fmt.Printf("  迭代 %d: goroutines=%d, 堆内存=%d KB\n",
				i, runtime.NumGoroutine(), ms.HeapInuse/1024)
		}
	}

	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	fmt.Printf("  最终: goroutines=%d, 堆内存=%d KB\n",
		runtime.NumGoroutine(), ms.HeapInuse/1024)

	// 正确做法
	fmt.Println("\n--- 正确做法: 始终 defer cancel() ---")
	for i := range 1000 {
		ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
		// 正确：立即 defer cancel()
		// 即使函数提前返回或 context 超时，都能释放 timer
		_ = ctx
		cancel() // 在循环中直接调用（非 defer，因为是循环体）

		if i%200 == 0 {
			var ms2 runtime.MemStats
			runtime.ReadMemStats(&ms2)
			fmt.Printf("  迭代 %d: goroutines=%d, 堆内存=%d KB\n",
				i, runtime.NumGoroutine(), ms2.HeapInuse/1024)
		}
	}

	fmt.Println("\n总结:")
	fmt.Println("  1. WithTimeout/WithDeadline 必须配对 cancel()，即使预期会自然超时")
	fmt.Println("  2. 函数内使用 defer cancel()；循环内直接调用 cancel()")
	fmt.Println("  3. cancel() 是幂等的，多次调用不会 panic")
}
