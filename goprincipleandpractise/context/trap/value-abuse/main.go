package main

import (
	"context"
	"fmt"
	"time"
)

/*
陷阱：context.Value 的常见滥用

运行：go run .

演示：
  1. 使用 string 作为 key 导致跨包命名冲突
  2. WithValue 链过深导致查找性能退化
  3. 正确做法：使用不可导出的类型作为 key + 结构体打包
*/

func main() {
	fmt.Println("=== 陷阱1: string key 跨包冲突 ===")
	demoStringKeyCollision()

	fmt.Println("\n=== 陷阱2: WithValue 链过深导致查找变慢 ===")
	demoDeepChainLookup()

	fmt.Println("\n=== 正确做法: 不可导出类型 + 结构体打包 ===")
	demoCorrectPattern()
}

func demoStringKeyCollision() {
	// 模拟两个不同的"包"都用 string "user_id" 作为 key
	ctx := context.Background()
	ctx = context.WithValue(ctx, "user_id", "package-A-value") // "包 A" 设置
	ctx = context.WithValue(ctx, "user_id", "package-B-value") // "包 B" 覆盖

	// 包 A 以为自己能取到 "package-A-value"，但实际取到的是 B 的值
	fmt.Printf("  ctx.Value(\"user_id\") = %v\n", ctx.Value("user_id"))
	fmt.Println("  包 A 的值被包 B 覆盖了！这就是 string key 的风险。")
	fmt.Println("  解决方案: 每个包定义自己的不可导出类型作为 key。")
}

func demoDeepChainLookup() {
	// 构造 100 层深的 WithValue 链
	ctx := context.Background()
	for i := range 100 {
		ctx = context.WithValue(ctx, i, i)
	}

	// 查找第 0 层的 key（需要遍历整条链）
	start := time.Now()
	iterations := 1_000_000
	for range iterations {
		_ = ctx.Value(0)
	}
	duration := time.Since(start)

	// 对比查找最近一层的 key
	start2 := time.Now()
	for range iterations {
		_ = ctx.Value(99)
	}
	duration2 := time.Since(start2)

	fmt.Printf("  查找最底层 key(0): %v / %d次\n", duration, iterations)
	fmt.Printf("  查找最顶层 key(99): %v / %d次\n", duration2, iterations)
	fmt.Printf("  深层查找耗时约为浅层的 %.1f 倍\n", float64(duration)/float64(duration2))
}

// 正确做法：不可导出类型 + 结构体打包

type requestMetaKey struct{} // 不可导出，其他包无法构造相同的 key

type RequestMeta struct {
	TraceID string
	Locale  string
	UserID  int64
}

func demoCorrectPattern() {
	meta := &RequestMeta{
		TraceID: "abc-123",
		UserID:  42,
		Locale:  "zh-CN",
	}
	ctx := context.WithValue(context.Background(), requestMetaKey{}, meta)

	// 一次查找获取所有数据
	if m, ok := ctx.Value(requestMetaKey{}).(*RequestMeta); ok {
		fmt.Printf("  TraceID=%s, UserID=%d, Locale=%s\n", m.TraceID, m.UserID, m.Locale)
	}
	fmt.Println("  使用 struct{} 类型作为 key，其他包无法构造相同的 key，彻底避免冲突。")
	fmt.Println("  所有 request-scoped 数据打包到一个结构体，只需一次 WithValue。")
}
