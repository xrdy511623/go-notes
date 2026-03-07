package parallel_vs_sequential

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// parallel_vs_sequential 对比集成测试并行执行和顺序执行的速度差异
//
// 在集成测试中，每个测试通常需要：
// 1. 准备数据（INSERT）
// 2. 执行操作
// 3. 验证结果（SELECT）
// 4. 清理数据（ROLLBACK / DELETE）
//
// 如果每个测试使用独立事务，它们可以安全地并行执行。
// 本 benchmark 对比了并行和顺序两种执行模式的耗时。

// TestCase 模拟一个集成测试用例
type TestCase struct {
	Name     string
	Duration time.Duration // 模拟测试执行耗时（含数据库 I/O）
}

// RunSequential 顺序执行所有测试用例
func RunSequential(cases []TestCase) time.Duration {
	start := time.Now()
	for _, tc := range cases {
		time.Sleep(tc.Duration)
	}
	return time.Since(start)
}

// RunParallel 并行执行所有测试用例
func RunParallel(cases []TestCase) time.Duration {
	start := time.Now()
	var wg sync.WaitGroup
	for _, tc := range cases {
		wg.Add(1)
		go func(d time.Duration) {
			defer wg.Done()
			time.Sleep(d)
		}(tc.Duration)
	}
	wg.Wait()
	return time.Since(start)
}

// GenerateTestCases 生成 n 个模拟测试用例
func GenerateTestCases(n int, avgDuration time.Duration) []TestCase {
	cases := make([]TestCase, n)
	for i := 0; i < n; i++ {
		cases[i] = TestCase{
			Name:     fmt.Sprintf("TestCase_%d", i),
			Duration: avgDuration,
		}
	}
	return cases
}

// SimulateDBOperation 模拟一个数据库操作（用于 benchmark）
func SimulateDBOperation(ctx context.Context, latency time.Duration) error {
	select {
	case <-time.After(latency):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
