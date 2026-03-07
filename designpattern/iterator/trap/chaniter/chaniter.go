package chaniter

import "runtime"

// ❌ 错误示例：用 channel + goroutine 实现迭代器
//
// 这是 Go 1.23 之前常见的"伪迭代器"模式。表面上可以用 for-range 遍历 channel，
// 看起来像迭代器，但有严重的缺陷。
//
// 问题清单：
//  1. goroutine 泄漏：消费方 break 后，生产方 goroutine 阻塞在 ch <- 上，永远无法退出
//  2. 无法传递错误：channel 只能传值，迭代中的错误没有标准方式返回
//  3. 性能开销：每次 send/recv 涉及 goroutine 调度，比函数调用慢 10-100 倍
//  4. 生命周期不可控：依赖 GC 回收泄漏的 goroutine，但 GC 不回收阻塞中的 goroutine
//
// 正确做法：见 iterator/sequence/ —— 使用 iter.Seq[V] 实现迭代器，
// 消费方 break 时 yield 返回 false，生产方立即停止，无 goroutine、无泄漏。

// Generate 用 channel 生成 0 到 n-1 的序列。
// ❌ 如果消费方提前 break，生产方 goroutine 永久阻塞。
func Generate(n int) <-chan int {
	ch := make(chan int)
	go func() {
		for i := 0; i < n; i++ {
			ch <- i // 消费方 break 后，这里永远阻塞
		}
		close(ch)
	}()
	return ch
}

// LeakedGoroutines 演示 goroutine 泄漏：
// 消费 3 个元素后 break，生产方 goroutine 永远阻塞在 ch <- 上。
func LeakedGoroutines() int {
	before := runtime.NumGoroutine()

	ch := Generate(1_000_000)
	count := 0
	for range ch {
		count++
		if count == 3 {
			break
		}
	}
	// 此时 Generate 内的 goroutine 阻塞在 ch <- 上，永远不会被 GC 回收。
	// runtime.NumGoroutine() 会比调用前多至少 1。

	after := runtime.NumGoroutine()
	return after - before // ≥ 1，泄漏的 goroutine
}
