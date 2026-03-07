package trap

import (
	"context"
	"fmt"
	"runtime"
	"time"
)

// ============================================================
// 泄漏模式1：忘记关闭channel导致goroutine永久阻塞
// ============================================================

// LeakForgotClose 生产者忘记关闭channel，消费者永远阻塞在 range
func LeakForgotClose() {
	ch := make(chan int)

	// 生产者：发送完数据但忘记close(ch)
	go func() {
		for i := 0; i < 5; i++ {
			ch <- i
		}
		// 缺少: close(ch) ← 导致泄漏
	}()

	// 消费者：range会永远等待，goroutine泄漏
	go func() {
		for v := range ch { // 永远不会退出
			_ = v
		}
	}()
}

// FixedForgotClose 修复：生产者用 defer close
func FixedForgotClose() <-chan int {
	ch := make(chan int)
	go func() {
		defer close(ch) // 正确：确保关闭
		for i := 0; i < 5; i++ {
			ch <- i
		}
	}()
	return ch
}

// ============================================================
// 泄漏模式2：无context取消的后台goroutine
// ============================================================

// LeakNoContext 启动后台goroutine但无法停止它
func LeakNoContext() {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for range ticker.C {
			// 永远运行，无法从外部停止
			fmt.Println("tick")
		}
	}()
}

// FixedWithContext 修复：通过context控制goroutine生命周期
func FixedWithContext(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				fmt.Println("tick")
			case <-ctx.Done():
				return // context取消时退出
			}
		}
	}()
}

// ============================================================
// 泄漏模式3：向无缓冲channel发送，但没有接收者
// ============================================================

// LeakNoReceiver 发送方阻塞在无缓冲channel上，永远等不到接收者
func LeakNoReceiver() {
	ch := make(chan string)
	go func() {
		ch <- "data" // 永远阻塞，因为没有接收者
	}()
	// 函数返回，ch的接收者不存在，goroutine泄漏
}

// FixedWithBufferOrContext 修复方案1：使用带缓冲channel
// 修复方案2：配合context超时
func FixedWithBufferOrContext(ctx context.Context) string {
	ch := make(chan string, 1) // 带缓冲，发送不阻塞
	go func() {
		ch <- "data"
	}()

	select {
	case v := <-ch:
		return v
	case <-ctx.Done():
		return ""
	}
}

// GoroutineCount 返回当前goroutine数量，用于检测泄漏
func GoroutineCount() int {
	return runtime.NumGoroutine()
}
