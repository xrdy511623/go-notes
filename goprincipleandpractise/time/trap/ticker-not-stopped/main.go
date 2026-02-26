package main

import (
	"fmt"
	"runtime"
	"time"
)

/*
陷阱：Ticker 不 Stop 导致 goroutine/内存泄漏

运行：go run .

预期行为：
  time.NewTicker 会在 runtime 中注册一个定时器，每隔 period 向 channel 发送一个值。
  如果不调用 ticker.Stop()，该定时器会一直存在于 runtime 的 timer 堆中。
  在 Go 1.23 之前，即使 Ticker 变量已无引用，GC 也不会回收它。

  在典型的 HTTP handler 或短生命周期函数中创建 Ticker 而不 Stop，
  会随着请求量增长不断泄漏 timer 资源。

  正确做法：始终 defer ticker.Stop()
*/

func main() {
	fmt.Println("=== 错误做法：Ticker 不 Stop ===")

	var memBefore, memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	// 模拟在循环/handler 中创建 Ticker 而不 Stop
	for i := 0; i < 10000; i++ {
		ticker := time.NewTicker(time.Hour) // 创建 Ticker
		_ = ticker                          // 不 Stop，不使用
		// 函数返回后 ticker 变量被丢弃，但 timer 仍在 runtime 中
	}

	runtime.GC()
	runtime.ReadMemStats(&memAfter)
	fmt.Printf("  创建 10000 个未 Stop 的 Ticker\n")
	fmt.Printf("  总分配字节: %d KB\n", (memAfter.TotalAlloc-memBefore.TotalAlloc)/1024)
	fmt.Printf("  总分配对象: %d\n", memAfter.Mallocs-memBefore.Mallocs)
	fmt.Printf("  goroutine 数: %d\n", runtime.NumGoroutine())

	fmt.Println("\n=== 正确做法：defer ticker.Stop() ===")
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	for i := 0; i < 10000; i++ {
		ticker := time.NewTicker(time.Hour)
		ticker.Stop() // 立即 Stop，释放 runtime timer 资源
	}

	runtime.GC()
	runtime.ReadMemStats(&memAfter)
	fmt.Printf("  创建并立即 Stop 10000 个 Ticker\n")
	fmt.Printf("  总分配字节: %d KB\n", (memAfter.TotalAlloc-memBefore.TotalAlloc)/1024)
	fmt.Printf("  总分配对象: %d\n", memAfter.Mallocs-memBefore.Mallocs)

	fmt.Println("\n=== 正确使用模式 ===")
	fmt.Println("func worker(ctx context.Context) {")
	fmt.Println("    ticker := time.NewTicker(time.Second)")
	fmt.Println("    defer ticker.Stop() // 必须 Stop！")
	fmt.Println("    for {")
	fmt.Println("        select {")
	fmt.Println("        case <-ctx.Done():")
	fmt.Println("            return")
	fmt.Println("        case <-ticker.C:")
	fmt.Println("            doWork()")
	fmt.Println("        }")
	fmt.Println("    }")
	fmt.Println("}")

	fmt.Println("\n总结:")
	fmt.Println("  1. Ticker 必须 Stop，否则 timer 资源泄漏")
	fmt.Println("  2. 在函数退出时 defer ticker.Stop()")
	fmt.Println("  3. Go 1.23+ 无引用的 Ticker 可被 GC，但最佳实践仍是显式 Stop")
}
