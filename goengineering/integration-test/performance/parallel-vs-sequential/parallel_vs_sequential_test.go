package parallel_vs_sequential

import (
	"testing"
	"time"
)

// 对比集成测试并行执行和顺序执行的速度差异
//
// 运行方式:
//   go test -run='^$' -bench=. -benchmem -benchtime=3s .
//
// 预期结果:
//   10 个测试用例（每个 1ms I/O），并行执行约 1ms，顺序执行约 10ms
//   并行加速比接近测试用例数量（受 CPU 核数限制）

func BenchmarkSequential_10Cases(b *testing.B) {
	cases := GenerateTestCases(10, 1*time.Millisecond)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RunSequential(cases)
	}
}

func BenchmarkParallel_10Cases(b *testing.B) {
	cases := GenerateTestCases(10, 1*time.Millisecond)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RunParallel(cases)
	}
}

func BenchmarkSequential_50Cases(b *testing.B) {
	cases := GenerateTestCases(50, 1*time.Millisecond)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RunSequential(cases)
	}
}

func BenchmarkParallel_50Cases(b *testing.B) {
	cases := GenerateTestCases(50, 1*time.Millisecond)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RunParallel(cases)
	}
}

func BenchmarkSequential_100Cases_5msIO(b *testing.B) {
	cases := GenerateTestCases(100, 5*time.Millisecond)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RunSequential(cases)
	}
}

func BenchmarkParallel_100Cases_5msIO(b *testing.B) {
	cases := GenerateTestCases(100, 5*time.Millisecond)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RunParallel(cases)
	}
}
