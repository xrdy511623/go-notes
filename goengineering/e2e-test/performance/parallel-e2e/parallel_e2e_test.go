package parallel_e2e

import (
	"testing"
	"time"
)

// 对比 E2E 测试并行执行和顺序执行的速度差异
//
// 运行方式:
//   go test -run='^$' -bench=. -benchmem -benchtime=5x .
//
// 预期结果:
//   5 个 E2E 测试（每个 3 步 × 2ms 延迟 = 6ms）：
//   - 顺序执行: ~30ms
//   - 并行执行: ~6ms（约 5x 加速）
//
//   10 个 E2E 测试（每个 5 步 × 5ms 延迟 = 25ms）：
//   - 顺序执行: ~250ms
//   - 并行执行: ~25ms（约 10x 加速）

// --- 场景一：5 个轻量级 E2E 测试 ---

func BenchmarkSequential_5Tests_Light(b *testing.B) {
	cases := GenerateE2ECases(5, 3, 2*time.Millisecond)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := RunSequential(cases); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParallel_5Tests_Light(b *testing.B) {
	cases := GenerateE2ECases(5, 3, 2*time.Millisecond)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := RunParallel(cases); err != nil {
			b.Fatal(err)
		}
	}
}

// --- 场景二：10 个标准 E2E 测试 ---

func BenchmarkSequential_10Tests_Standard(b *testing.B) {
	cases := GenerateE2ECases(10, 5, 5*time.Millisecond)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := RunSequential(cases); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParallel_10Tests_Standard(b *testing.B) {
	cases := GenerateE2ECases(10, 5, 5*time.Millisecond)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := RunParallel(cases); err != nil {
			b.Fatal(err)
		}
	}
}

// --- 场景三：20 个重量级 E2E 测试（模拟真实场景） ---

func BenchmarkSequential_20Tests_Heavy(b *testing.B) {
	cases := GenerateE2ECases(20, 8, 10*time.Millisecond)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := RunSequential(cases); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParallel_20Tests_Heavy(b *testing.B) {
	cases := GenerateE2ECases(20, 8, 10*time.Millisecond)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := RunParallel(cases); err != nil {
			b.Fatal(err)
		}
	}
}
