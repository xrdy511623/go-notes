package performance

import (
	"context"
	"fmt"
	"testing"
)

/*
Context 创建与取消传播性能

执行命令:

	go test -run '^$' -bench 'Cancel|Creation' -benchmem .

对比维度:
  1. Context 创建开销: WithCancel vs WithTimeout vs WithValue vs WithCancelCause
  2. Cancel 传播: 不同子节点数量下 cancel() 的耗时
  3. Cancel 传播: 不同链深度下根节点 cancel() 的耗时

结论:
  - WithCancel 创建最轻量（~100ns），WithTimeout 需额外创建 Timer（~300ns）
  - WithValue 不涉及 cancel 机制，创建最快（~50ns）
  - WithCancelCause 与 WithCancel 开销接近
  - cancel() 的耗时与子节点数量成正比（遍历 children map + 递归 cancel）
  - 深链 cancel 也是线性的，但每层只有一个子节点，开销比宽树小
*/

// ------------------- Context 创建开销 -------------------

func BenchmarkCreationCancel(b *testing.B) {
	for b.Loop() {
		_, cancel := ContextCreationCancel()
		cancel()
	}
}

func BenchmarkCreationTimeout(b *testing.B) {
	for b.Loop() {
		_, cancel := ContextCreationTimeout()
		cancel()
	}
}

func BenchmarkCreationValue(b *testing.B) {
	for b.Loop() {
		cancelSink = ContextCreationValue()
	}
}

func BenchmarkCreationCancelCause(b *testing.B) {
	for b.Loop() {
		_, cancel := ContextCreationCancelCause()
		cancel(nil)
	}
}

// ------------------- Cancel 传播（宽树） -------------------

func BenchmarkCancelPropagationWide(b *testing.B) {
	for _, n := range []int{1, 10, 100, 1000} {
		b.Run(fmt.Sprintf("children=%d", n), func(b *testing.B) {
			for b.Loop() {
				_, cancel := CreateCancelTree(n)
				cancel()
			}
		})
	}
}

// ------------------- Cancel 传播（深链） -------------------

func BenchmarkCancelPropagationDeep(b *testing.B) {
	for _, d := range []int{1, 10, 100} {
		b.Run(fmt.Sprintf("depth=%d", d), func(b *testing.B) {
			for b.Loop() {
				_, cancel := CreateDeepCancelChain(d)
				cancel()
			}
		})
	}
}

// ------------------- Done() channel 检查开销 -------------------

func BenchmarkDoneCheck(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b.ResetTimer()
	for b.Loop() {
		select {
		case <-ctx.Done():
		default:
		}
	}
}

func BenchmarkDoneCheckBackground(b *testing.B) {
	ctx := context.Background()
	for b.Loop() {
		select {
		case <-ctx.Done():
		default:
		}
	}
}
