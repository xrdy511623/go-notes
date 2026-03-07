// Package main 演示 goroutine 泄露检测
//
// 测试中启动 goroutine 但未等待其完成，会导致：
// 1. 测试看起来通过了，但 goroutine 还在后台运行
// 2. goroutine 的 panic 不会被测试捕获
// 3. 持续累积的 goroutine 消耗内存
//
// 检测方法：比较测试前后的 runtime.NumGoroutine()
//
// 运行方式：go run ./goprincipleandpractise/unit-test/trap/goroutine-leak/
package main

import (
	"fmt"
	"runtime"
	"time"
)

func main() {
	fmt.Println("=== Goroutine 泄露检测演示 ===")
	fmt.Println()

	// 记录初始 goroutine 数量
	initial := runtime.NumGoroutine()
	fmt.Printf("初始 goroutine 数量: %d\n", initial)

	// ❌ 错误：启动 goroutine 但不等待完成
	fmt.Println("\n--- 泄露场景 ---")
	for i := 0; i < 5; i++ {
		go func(n int) {
			// 模拟长时间工作
			time.Sleep(10 * time.Second)
			fmt.Printf("  goroutine %d completed (you won't see this)\n", n)
		}(i)
	}

	// 不等待，直接检查
	time.Sleep(100 * time.Millisecond) // 让 goroutine 启动
	leaked := runtime.NumGoroutine()
	fmt.Printf("当前 goroutine 数量: %d (泄露了 %d 个)\n", leaked, leaked-initial)

	// ✅ 正确：使用 channel 或 WaitGroup 等待
	fmt.Println("\n--- 正确做法 ---")
	fmt.Println("  1. 使用 sync.WaitGroup 等待所有 goroutine")
	fmt.Println("  2. 使用 context.WithCancel 通知 goroutine 退出")
	fmt.Println("  3. 使用 goleak 库自动检测：")
	fmt.Println("     go.uber.org/goleak")
	fmt.Println()

	fmt.Println("--- 测试中的检测模式 ---")
	codeExample := []string{
		"  func TestNoLeak(t *testing.T) {",
		"      before := runtime.NumGoroutine()",
		"",
		"      // ... 执行被测代码 ...",
		"",
		"      // 给 goroutine 一点时间结束",
		"      time.Sleep(100 * time.Millisecond)",
		"",
		"      after := runtime.NumGoroutine()",
		"      if after > before {",
		`          t.Errorf("goroutine leak: before=%d, after=%d", before, after)`,
		"      }",
		"  }",
	}
	for _, line := range codeExample {
		fmt.Println(line)
	}

	fmt.Println("\n注意：本示例故意泄露 goroutine 以演示问题，程序退出时它们会被回收。")
}
