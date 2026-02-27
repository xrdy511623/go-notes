package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

/*
陷阱：用各种方式掩盖不稳定测试，而不是修复根因

运行：go run .

预期行为：
  模拟一个因时序问题导致的 Flaky Test。
  演示三种错误"解决"方式和一种正确方式，对比它们的行为差异。

  正确做法：分析不稳定根因（时序？数据冲突？资源竞争？），针对性修复。
*/

// simulateFlakyService 模拟一个不稳定的服务调用
// 有 40% 概率因"超时"而失败
func simulateFlakyService() error {
	latency := time.Duration(rand.Intn(100)) * time.Millisecond
	time.Sleep(latency)
	if latency > 60*time.Millisecond {
		return fmt.Errorf("request timeout after %v", latency)
	}
	return nil
}

// simulateStableServiceWithPoll 模拟稳定的服务调用（用轮询代替固定等待）
func simulateStableServiceWithPoll(maxRetries int) error {
	for i := 0; i < maxRetries; i++ {
		latency := time.Duration(rand.Intn(100)) * time.Millisecond
		if latency <= 60*time.Millisecond {
			return nil // 成功
		}
		// 等待后重试，但限制次数
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("still failing after %d retries", maxRetries)
}

func main() {
	rand.New(rand.NewSource(time.Now().UnixNano()))

	fmt.Println("=== 模拟 Flaky Test 的不同处理方式 ===")
	fmt.Println()

	const totalRuns = 20

	// ❌ 做法一：直接跑，失败就失败（不处理 flaky）
	fmt.Println("--- ❌ 做法一：不处理，随缘 ---")
	passed, failed := 0, 0
	for i := 0; i < totalRuns; i++ {
		if err := simulateFlakyService(); err != nil {
			failed++
		} else {
			passed++
		}
	}
	fmt.Printf("  结果: %d/%d 通过 (通过率 %.0f%%)\n", passed, totalRuns, float64(passed)/float64(totalRuns)*100)
	fmt.Println("  问题: 测试时好时坏，团队逐渐不信任测试套件")
	fmt.Println()

	// ❌ 做法二：无限重试直到通过
	fmt.Println("--- ❌ 做法二：无限重试直到通过 ---")
	start := time.Now()
	retryCount := 0
	for {
		retryCount++
		if err := simulateFlakyService(); err == nil {
			break
		}
	}
	fmt.Printf("  重试 %d 次后通过，耗时 %v\n", retryCount, time.Since(start).Round(time.Millisecond))
	fmt.Println("  问题: 永远不会失败，掩盖了真实问题；如果服务真挂了，测试会死循环")
	fmt.Println()

	// ❌ 做法三：t.Skip 跳过
	fmt.Println("--- ❌ 做法三：t.Skip 跳过 ---")
	fmt.Println("  // t.Skip(\"flaky test, skipping for now\")")
	fmt.Println("  结果: 测试永远不运行，覆盖的功能失去保护")
	fmt.Println("  \"for now\" 在代码库中意味着 forever")
	fmt.Println()

	// ✅ 正确做法：限制重试 + 告警 + 记录统计
	fmt.Println("--- ✅ 正确做法：限制重试 + 统计通过率 + 修复根因 ---")
	const maxRetries = 3
	passedGood, failedGood := 0, 0
	retryUsed := 0
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < totalRuns; i++ {
		wg.Add(1)
		go func(run int) {
			defer wg.Done()
			var lastErr error
			for attempt := 0; attempt < maxRetries; attempt++ {
				lastErr = simulateFlakyService()
				if lastErr == nil {
					mu.Lock()
					passedGood++
					if attempt > 0 {
						retryUsed++
					}
					mu.Unlock()
					return
				}
			}
			mu.Lock()
			failedGood++
			mu.Unlock()
		}(i)
	}
	wg.Wait()

	passRate := float64(passedGood) / float64(totalRuns) * 100
	fmt.Printf("  结果: %d/%d 通过 (通过率 %.0f%%)\n", passedGood, totalRuns, passRate)
	fmt.Printf("  其中 %d 次需要重试才通过（这些是需要修复的信号）\n", retryUsed)
	if failedGood > 0 {
		fmt.Printf("  %d 次在 %d 次重试后仍然失败（标记为需修复）\n", failedGood, maxRetries)
	}
	fmt.Println()

	if passRate < 95 {
		fmt.Println("  ⚠ 通过率 < 95%，必须修复根因：")
	}
	fmt.Println("  修复步骤:")
	fmt.Println("    1. 分析失败原因：是超时？数据冲突？资源竞争？")
	fmt.Println("    2. 本例中：服务延迟有 40% 概率超过阈值")
	fmt.Println("    3. 正确修复：用轮询等待替代固定超时阈值")
	fmt.Println()

	// 演示修复后的效果
	fmt.Println("--- ✅ 修复根因后的效果 ---")
	passedFixed := 0
	for i := 0; i < totalRuns; i++ {
		if err := simulateStableServiceWithPoll(5); err == nil {
			passedFixed++
		}
	}
	fmt.Printf("  结果: %d/%d 通过 (通过率 %.0f%%)\n",
		passedFixed, totalRuns, float64(passedFixed)/float64(totalRuns)*100)
	fmt.Println("  根因修复后，无需重试机制，测试天然稳定")
}
