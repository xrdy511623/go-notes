package parallel_e2e

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// parallel_e2e 对比 E2E 测试并行执行和顺序执行的速度差异
//
// E2E 测试通常是 I/O 密集型（HTTP 请求、等待响应），
// 如果测试之间相互独立（数据隔离），完全可以并行执行。
//
// 本 benchmark 模拟多个独立的 E2E 测试用例，对比两种执行模式的耗时。

// E2ETestCase 模拟一个 E2E 测试用例
type E2ETestCase struct {
	Name      string
	Steps     int           // 测试步骤数
	StepDelay time.Duration // 每个步骤的延迟（模拟 HTTP 往返）
}

// Run 执行测试用例
func (tc *E2ETestCase) Run() error {
	for i := 0; i < tc.Steps; i++ {
		// 模拟 HTTP 请求延迟（加入少量随机性模拟真实网络）
		jitter := time.Duration(rand.Intn(int(tc.StepDelay / 5)))
		time.Sleep(tc.StepDelay + jitter)
	}
	return nil
}

// RunSequential 顺序执行所有 E2E 测试
func RunSequential(cases []E2ETestCase) (time.Duration, error) {
	start := time.Now()
	for i := range cases {
		if err := cases[i].Run(); err != nil {
			return time.Since(start), fmt.Errorf("test %s failed: %w", cases[i].Name, err)
		}
	}
	return time.Since(start), nil
}

// RunParallel 并行执行所有 E2E 测试
func RunParallel(cases []E2ETestCase) (time.Duration, error) {
	start := time.Now()
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
	)

	for i := range cases {
		wg.Add(1)
		go func(tc *E2ETestCase) {
			defer wg.Done()
			if err := tc.Run(); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("test %s failed: %w", tc.Name, err)
				}
				mu.Unlock()
			}
		}(&cases[i])
	}

	wg.Wait()
	return time.Since(start), firstErr
}

// GenerateE2ECases 生成模拟 E2E 测试用例
func GenerateE2ECases(count, stepsPerTest int, stepDelay time.Duration) []E2ETestCase {
	cases := make([]E2ETestCase, count)
	for i := 0; i < count; i++ {
		cases[i] = E2ETestCase{
			Name:      fmt.Sprintf("E2E_Test_%d", i),
			Steps:     stepsPerTest,
			StepDelay: stepDelay,
		}
	}
	return cases
}
