// Package main 演示循环变量捕获陷阱
//
// 在 Go 1.21 及之前，for range 循环变量在所有迭代中共享同一个地址，
// 闭包捕获的是变量的引用而非值。这会导致所有 goroutine/子测试看到最后一个迭代值。
//
// Go 1.22+ 改变了语义：每次迭代创建新的变量，此问题不再存在。
// 本仓库使用 Go 1.24，但为教学目的保留此演示。
//
// 运行方式：go run ./goprincipleandpractise/unit-test/trap/loop-capture/
package main

import (
	"fmt"
	"sync"
)

func main() {
	fmt.Println("=== 循环变量捕获陷阱演示 ===")
	fmt.Println()

	// ❌ 错误写法（Go 1.21 及之前的 bug）
	// 在 Go 1.22+ 中此代码实际上是正确的，
	// 但在旧版本中所有 goroutine 都会打印 "Charlie"
	fmt.Println("--- 旧版 Go 中的问题 ---")
	names := []string{"Alice", "Bob", "Charlie"}

	var wg sync.WaitGroup
	for _, name := range names {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Go 1.21-: name 始终是 "Charlie"（最后一个值）
			// Go 1.22+: name 正确捕获每次迭代的值
			fmt.Printf("  Hello, %s\n", name)
		}()
	}
	wg.Wait()

	fmt.Println()

	// ✅ 兼容所有版本的正确写法
	fmt.Println("--- 安全写法（兼容所有 Go 版本） ---")
	for _, name := range names {
		name := name // 显式捕获（Go 1.22+ 中多余但无害）
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Printf("  Hello, %s\n", name)
		}()
	}
	wg.Wait()

	fmt.Println()

	// 在测试中的典型表现
	fmt.Println("--- 测试中的典型问题 ---")
	fmt.Println("  在旧版 Go 中，表驱动测试 + t.Parallel() 容易触发此 bug：")
	fmt.Println("  所有子测试都使用 testCases 的最后一个元素。")
	fmt.Println("  解决方案：tt := tt 或升级到 Go 1.22+")
	fmt.Println("  详见：parallel_test.go")
}
