package poll_vs_sleep

import (
	"testing"
	"time"
)

// 对比固定 Sleep 和轮询等待的耗时差异
//
// 运行方式:
//   go test -run='^$' -bench=. -benchmem -benchtime=10x .
//
// 预期结果:
//   服务启动需要 50ms 时：
//   - 固定 Sleep 200ms → 每次都等 200ms（浪费 150ms）
//   - 轮询 10ms 间隔   → 约 50-60ms 就能检测到就绪
//
//   轮询方式平均比固定 Sleep 快 3-4 倍

// BenchmarkSleep_200ms 固定等待 200ms（服务实际 50ms 就绪）
func BenchmarkSleep_200ms(b *testing.B) {
	for i := 0; i < b.N; i++ {
		svc := NewServiceSimulator(50 * time.Millisecond)
		_ = WaitWithSleep(200 * time.Millisecond)
		_ = svc // 服务早就就绪了，但我们还在等
	}
}

// BenchmarkPoll_10msInterval 轮询间隔 10ms（服务实际 50ms 就绪）
func BenchmarkPoll_10msInterval(b *testing.B) {
	for i := 0; i < b.N; i++ {
		svc := NewServiceSimulator(50 * time.Millisecond)
		_, err := WaitWithPoll(svc, 10*time.Millisecond, 5*time.Second)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSleep_500ms 固定等待 500ms（服务实际 200ms 就绪）
func BenchmarkSleep_500ms(b *testing.B) {
	for i := 0; i < b.N; i++ {
		svc := NewServiceSimulator(200 * time.Millisecond)
		_ = WaitWithSleep(500 * time.Millisecond)
		_ = svc
	}
}

// BenchmarkPoll_50msInterval 轮询间隔 50ms（服务实际 200ms 就绪）
func BenchmarkPoll_50msInterval(b *testing.B) {
	for i := 0; i < b.N; i++ {
		svc := NewServiceSimulator(200 * time.Millisecond)
		_, err := WaitWithPoll(svc, 50*time.Millisecond, 5*time.Second)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSleep_2s 固定等待 2s（服务实际 500ms 就绪）
func BenchmarkSleep_2s(b *testing.B) {
	for i := 0; i < b.N; i++ {
		svc := NewServiceSimulator(500 * time.Millisecond)
		_ = WaitWithSleep(2 * time.Second)
		_ = svc
	}
}

// BenchmarkPoll_100msInterval 轮询间隔 100ms（服务实际 500ms 就绪）
func BenchmarkPoll_100msInterval(b *testing.B) {
	for i := 0; i < b.N; i++ {
		svc := NewServiceSimulator(500 * time.Millisecond)
		_, err := WaitWithPoll(svc, 100*time.Millisecond, 5*time.Second)
		if err != nil {
			b.Fatal(err)
		}
	}
}
