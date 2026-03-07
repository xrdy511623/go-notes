package pattern

import (
	"context"
	"sort"
	"testing"
)

func heavyCompute(n int) int {
	// 模拟CPU密集计算
	result := n
	for i := 0; i < 100; i++ {
		result = result*31 + i
	}
	return result
}

func TestFanOutFanIn(t *testing.T) {
	ctx := context.Background()
	in := Generator(ctx, 1, 2, 3, 4, 5, 6, 7, 8)

	result := Collect(FanOutFanIn(ctx, in, 3, func(n int) int { return n * n }))

	// fan-out结果顺序不确定，排序后比较
	sort.Ints(result)
	expected := []int{1, 4, 9, 16, 25, 36, 49, 64}

	if len(result) != len(expected) {
		t.Fatalf("got %d results, want %d", len(result), len(expected))
	}
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("result[%d] = %d, want %d", i, v, expected[i])
		}
	}
}

func TestFanOutFanInCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	nums := make([]int, 10000)
	for i := range nums {
		nums[i] = i
	}
	in := Generator(ctx, nums...)
	out := FanOutFanIn(ctx, in, 4, func(n int) int { return n * n })

	// 只取几个值就取消
	count := 0
	for range out {
		count++
		if count >= 5 {
			cancel()
			break
		}
	}
	// drain
	for range out {
	}
}

/*
Fan-out吞吐量基准：对比不同worker数量。

执行命令:

	go test -run '^$' -bench '^BenchmarkFanOut' -benchtime=3s -count=3 -benchmem .
*/

func BenchmarkFanOutWorkers1(b *testing.B) {
	benchFanOut(b, 1)
}

func BenchmarkFanOutWorkers4(b *testing.B) {
	benchFanOut(b, 4)
}

func BenchmarkFanOutWorkers8(b *testing.B) {
	benchFanOut(b, 8)
}

func benchFanOut(b *testing.B, workers int) {
	b.Helper()
	ctx := context.Background()
	nums := make([]int, 100)
	for i := range nums {
		nums[i] = i
	}
	for b.Loop() {
		in := Generator(ctx, nums...)
		Collect(FanOutFanIn(ctx, in, workers, heavyCompute))
	}
}
