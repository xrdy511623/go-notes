package pattern

import (
	"context"
	"runtime"
	"testing"
)

func TestWorkerPool(t *testing.T) {
	ctx := context.Background()
	jobs := Generator(ctx, 1, 2, 3, 4, 5)
	results := Collect(WorkerPool(ctx, jobs, 3, func(n int) int { return n * 10 }))

	if len(results) != 5 {
		t.Fatalf("got %d results, want 5", len(results))
	}

	// 结果顺序不确定，验证所有期望值都存在
	seen := make(map[int]bool)
	for _, v := range results {
		seen[v] = true
	}
	for _, expected := range []int{10, 20, 30, 40, 50} {
		if !seen[expected] {
			t.Errorf("missing expected value %d", expected)
		}
	}
}

/*
Worker Pool vs 无限goroutine：对比固定pool和每job一个goroutine的开销。

执行命令:

	go test -run '^$' -bench '^BenchmarkPool' -benchtime=3s -count=3 -benchmem .

关注指标:
  - ns/op: 总处理延迟
  - B/op: 内存分配（无限goroutine会分配更多栈空间）
  - allocs/op: 分配次数

预期结论:
 1. 任务数少时差异不大
 2. 任务数增大后，Worker Pool的内存开销稳定，无限goroutine线性增长
 3. Worker Pool通过channel的背压机制自动限流
*/

func BenchmarkPoolWorkerPool(b *testing.B) {
	ctx := context.Background()
	workers := runtime.GOMAXPROCS(0)
	nums := make([]int, 1000)
	for i := range nums {
		nums[i] = i
	}

	for b.Loop() {
		jobs := Generator(ctx, nums...)
		Collect(WorkerPool(ctx, jobs, workers, heavyCompute))
	}
}

func BenchmarkPoolUnbounded(b *testing.B) {
	ctx := context.Background()
	nums := make([]int, 1000)
	for i := range nums {
		nums[i] = i
	}

	for b.Loop() {
		Collect(UnboundedGoroutines(ctx, nums, heavyCompute))
	}
}
