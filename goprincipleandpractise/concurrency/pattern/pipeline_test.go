package pattern

import (
	"context"
	"testing"
)

func TestPipelineSquare(t *testing.T) {
	ctx := context.Background()
	ch := Generator(ctx, 2, 3, 4)
	result := Collect(Square(ctx, ch))

	expected := []int{4, 9, 16}
	if len(result) != len(expected) {
		t.Fatalf("got %d results, want %d", len(result), len(expected))
	}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("result[%d] = %d, want %d", i, v, expected[i])
		}
	}
}

func TestPipelineChained(t *testing.T) {
	ctx := context.Background()
	// 2,3,4 → square → 4,9,16 → double → 8,18,32
	ch := Generator(ctx, 2, 3, 4)
	result := Collect(Double(ctx, Square(ctx, ch)))

	expected := []int{8, 18, 32}
	if len(result) != len(expected) {
		t.Fatalf("got %d results, want %d", len(result), len(expected))
	}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("result[%d] = %d, want %d", i, v, expected[i])
		}
	}
}

func TestPipelineCancelEarly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// 生成大量数据，但立即取消
	nums := make([]int, 10000)
	for i := range nums {
		nums[i] = i
	}
	ch := Generator(ctx, nums...)
	out := Square(ctx, ch)

	// 只取第一个值就取消
	v := <-out
	if v != 0 {
		t.Errorf("first value = %d, want 0", v)
	}
	cancel()

	// 确认channel最终被关闭（goroutine不泄漏）
	for range out {
		// drain remaining
	}
}

/*
Pipeline吞吐量基准：对比不同阶段数的开销。

执行命令:

	go test -run '^$' -bench '^BenchmarkPipeline' -benchtime=3s -count=3 -benchmem .
*/

func BenchmarkPipelineSingleStage(b *testing.B) {
	ctx := context.Background()
	for b.Loop() {
		ch := Generator(ctx, 1, 2, 3, 4, 5)
		Collect(Square(ctx, ch))
	}
}

func BenchmarkPipelineTwoStages(b *testing.B) {
	ctx := context.Background()
	for b.Loop() {
		ch := Generator(ctx, 1, 2, 3, 4, 5)
		Collect(Double(ctx, Square(ctx, ch)))
	}
}

func BenchmarkPipelineThreeStages(b *testing.B) {
	ctx := context.Background()
	for b.Loop() {
		ch := Generator(ctx, 1, 2, 3, 4, 5)
		Collect(Filter(ctx, Double(ctx, Square(ctx, ch)), func(n int) bool { return n > 10 }))
	}
}
