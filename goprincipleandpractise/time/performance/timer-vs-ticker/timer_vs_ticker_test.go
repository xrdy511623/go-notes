package timervsticker

import (
	"testing"
	"time"
)

/*
对比循环创建 Timer vs 复用 Ticker/Timer 的性能差异。

执行命令:

	go test -run '^$' -bench '^Benchmark' -benchtime=3s -count=5 -benchmem .

预期结果:
  - NewTimer 每次循环：每次分配新 Timer 对象，allocs/op 高
  - Ticker/Timer Reset：复用同一个定时器，allocs/op 接近 0
*/

func BenchmarkNewTimerPerIteration(b *testing.B) {
	for b.Loop() {
		timer := time.NewTimer(time.Hour)
		timer.Stop()
	}
}

func BenchmarkTimerReuse(b *testing.B) {
	timer := time.NewTimer(time.Hour)
	defer timer.Stop()

	b.ResetTimer()
	for b.Loop() {
		timer.Reset(time.Hour)
	}
}

func BenchmarkTickerReuse(b *testing.B) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	b.ResetTimer()
	for b.Loop() {
		ticker.Reset(time.Hour)
	}
}

func BenchmarkNewTimerBatch100(b *testing.B) {
	for b.Loop() {
		SimulateTimerLoop(100)
	}
}

func BenchmarkTimerReuseBatch100(b *testing.B) {
	for b.Loop() {
		SimulateTimerReuse(100)
	}
}

func BenchmarkTickerReuseBatch100(b *testing.B) {
	for b.Loop() {
		SimulateTickerReuse(100)
	}
}
