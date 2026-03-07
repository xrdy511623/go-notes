package performance

import (
	"context"
	"fmt"
	"testing"
)

/*
context.Value 链表查找性能对比

执行命令:

	go test -run '^$' -bench 'Value' -benchmem .

对比维度:
  1. 不同链深度下 Value() 查找最底层 key 的耗时（O(n) 退化）
  2. 链式 WithValue（5 个 key）vs 结构体打包（1 次 WithValue）

结论:
  - Value() 查找是 O(n)，n 为从当前节点到目标节点的 context 链长度
  - 深度为 1 时约 3-5ns，深度为 100 时约 200-400ns
  - 将多个值打包到一个结构体中只需一次 WithValue，查找始终是 O(1)
  - 生产建议: 每个 middleware 不要各自 WithValue，应统一使用 request-scoped 结构体
*/

var depths = []int{1, 5, 10, 50, 100}

func BenchmarkValueLookupByDepth(b *testing.B) {
	for _, depth := range depths {
		ctx, targetKey := CreateValueChain(depth)
		b.Run(fmt.Sprintf("depth=%d", depth), func(b *testing.B) {
			for b.Loop() {
				valueSink = ctx.Value(targetKey)
			}
		})
	}
}

// BenchmarkValueLookupChain 链式 WithValue: 查找第 1 个插入的 key（最深）
func BenchmarkValueLookupChain(b *testing.B) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, benchKey(1), "traceID")
	ctx = context.WithValue(ctx, benchKey(2), int64(42))
	ctx = context.WithValue(ctx, benchKey(3), "tenantXYZ")
	ctx = context.WithValue(ctx, benchKey(4), "req001")
	ctx = context.WithValue(ctx, benchKey(5), "zhCN")
	b.ResetTimer()
	for b.Loop() {
		valueSink = ctx.Value(benchKey(1)) // 查找最深的 key
	}
}

// BenchmarkValueLookupStruct 结构体打包: 一次 WithValue + 字段访问
func BenchmarkValueLookupStruct(b *testing.B) {
	ctx := CreateStructValue()
	b.ResetTimer()
	for b.Loop() {
		meta := LookupStruct(ctx)
		valueSink = meta.TraceID
	}
}
