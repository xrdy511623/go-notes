package goroutineleak

import "fmt"

// 管道模式的典型陷阱：忘记取消导致 goroutine 泄漏
//
// 当管道消费者只需要部分数据时，如果不通知上游停止，
// 上游 goroutine 会永远阻塞在 channel 发送上，造成内存泄漏。

// ❌ 错误写法：没有取消机制
func leakyGenerator(nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for _, n := range nums {
			out <- n // ⚠️ 如果下游不再读取，这里永远阻塞
		}
	}()
	return out
}

// LeakyPipeline 演示 goroutine 泄漏：
// generator 发送了 5 个值，但消费者只读了 1 个就返回。
// generator 的 goroutine 阻塞在发送第 2 个值上，永远不会退出。
// 每调用一次 LeakyPipeline，runtime.NumGoroutine() 就增加 1。
func LeakyPipeline() {
	ch := leakyGenerator(1, 2, 3, 4, 5)
	first := <-ch
	fmt.Printf("只取了第一个值: %d，但 generator goroutine 已泄漏\n", first)
	// ch 没有被 drain，也没有取消机制
	// leakyGenerator 中的 goroutine 永远阻塞在 out <- 2
}

// ✅ 正确写法：用 context 取消通知上游
//
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()      // ← 消费者结束时取消，上游 goroutine 自动退出
//
//	ch := stage.Generate(ctx, 1, 2, 3, 4, 5)
//	first := <-ch       // 只取需要的数据
//	cancel()            // 通知上游停止
//	// Generate 的 goroutine 检测到 ctx.Done()，正常退出，无泄漏

// ❌ 错误写法二：fan-out 不 merge，忘记消费所有 worker 的输出
func leakyFanOut() {
	ch := leakyGenerator(1, 2, 3, 4, 5, 6, 7, 8)
	// 启动 4 个 worker，每个 worker 的输出 channel 没人消费
	for range 4 {
		out := make(chan int)
		go func() {
			defer close(out)
			for v := range ch {
				out <- v * v // ⚠️ 没有消费者读 out，goroutine 永远阻塞
			}
		}()
		// out 被丢弃，worker goroutine 泄漏
	}
}
