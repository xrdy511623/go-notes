package trap

import (
	"context"
	"fmt"
	"runtime"
	"time"
)

// RunAllTraps 演示所有并发陷阱（供外部go run调用）
func RunAllTraps() {
	trapGoroutineLeak()
	trapTimeSleepSync()
	trapLoopVarCapture()
}

// ============================================================
// 陷阱1：goroutine泄漏检测
// ============================================================

func trapGoroutineLeak() {
	fmt.Println("=== 陷阱1：goroutine泄漏 ===")

	before := runtime.NumGoroutine()
	fmt.Println("泄漏前 goroutine数:", before)

	// 制造3个泄漏
	LeakForgotClose()
	LeakNoReceiver()
	// LeakNoContext() 不调用，因为会持续打印

	time.Sleep(100 * time.Millisecond) // 等goroutine启动

	after := runtime.NumGoroutine()
	fmt.Println("泄漏后 goroutine数:", after)
	fmt.Printf("泄漏了 %d 个goroutine\n", after-before)

	// 正确的做法：使用context + defer close
	fmt.Println("正确做法: 用context控制生命周期, 用defer close(ch)关闭channel")
	fmt.Println()
}

// ============================================================
// 陷阱2：用time.Sleep做同步
// ============================================================

func trapTimeSleepSync() {
	fmt.Println("=== 陷阱2：time.Sleep不是同步原语 ===")

	// 错误示范（简化版，不会真的race因为是演示）
	var data string

	go func() {
		time.Sleep(50 * time.Millisecond) // 模拟一些工作
		data = "ready"
	}()

	// time.Sleep不建立happens-before关系，这是数据竞争
	time.Sleep(100 * time.Millisecond)
	fmt.Println("Sleep同步 (不可靠):", data)

	// 正确做法：用channel
	data = ""
	done := make(chan struct{})
	go func() {
		time.Sleep(50 * time.Millisecond)
		data = "ready"
		close(done)
	}()

	<-done // happens-before保证
	fmt.Println("Channel同步 (可靠):", data)
	fmt.Println()
}

// ============================================================
// 陷阱3：循环变量捕获（Go 1.22前的经典问题）
// Go 1.22+ 已修复此问题（每次迭代创建新变量），
// 但了解它仍然重要，因为旧代码库中大量存在。
// ============================================================

func trapLoopVarCapture() {
	fmt.Println("=== 陷阱3：循环变量捕获 ===")
	fmt.Println("注意: Go 1.22+ 已修复此问题，每次迭代创建新变量")

	// Go 1.22之前的问题：所有goroutine捕获同一个变量i
	// 修复方式1：传参（适用于所有Go版本）
	results := make(chan string, 5)
	for i := 0; i < 5; i++ {
		go func(idx int) { // 通过参数传入，创建副本
			results <- fmt.Sprintf("worker-%d", idx)
		}(i)
	}

	for j := 0; j < 5; j++ {
		fmt.Println(<-results)
	}

	// 修复方式2：局部变量（适用于所有Go版本）
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for i := 0; i < 3; i++ {
		i := i // 创建局部副本（Go 1.22前必需）
		go func() {
			select {
			case <-ctx.Done():
			default:
				_ = i
			}
		}()
	}

	fmt.Println("修复: 通过参数传入或创建局部变量")
	fmt.Println()
}
