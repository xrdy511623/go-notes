package sleepvstimer

import (
	"testing"
	"time"
)

/*
对比 time.Sleep vs time.NewTimer 的性能差异。

执行命令:

	go test -run '^$' -bench '^Benchmark' -benchtime=3s -count=5 -benchmem .

说明:
  time.Sleep 和 time.NewTimer 底层都使用 runtime timer。
  区别在于 Timer 额外创建 channel 和 Timer 对象。
  对于纯等待场景，Sleep 更轻量；Timer 的优势在于可用于 select。
*/

func BenchmarkSleep1us(b *testing.B) {
	for b.Loop() {
		time.Sleep(time.Microsecond)
	}
}

func BenchmarkTimer1us(b *testing.B) {
	for b.Loop() {
		timer := time.NewTimer(time.Microsecond)
		<-timer.C
	}
}

func BenchmarkSleep100us(b *testing.B) {
	for b.Loop() {
		time.Sleep(100 * time.Microsecond)
	}
}

func BenchmarkTimer100us(b *testing.B) {
	for b.Loop() {
		timer := time.NewTimer(100 * time.Microsecond)
		<-timer.C
	}
}

// BenchmarkTimerCreateStop measures Timer creation + Stop overhead
// without actually waiting (isolates allocation cost).
func BenchmarkTimerCreateStop(b *testing.B) {
	for b.Loop() {
		timer := time.NewTimer(time.Hour)
		timer.Stop()
	}
}

// BenchmarkTimerReuse measures the cost of resetting a reused Timer.
func BenchmarkTimerReuse(b *testing.B) {
	timer := time.NewTimer(time.Hour)
	defer timer.Stop()
	b.ResetTimer()
	for b.Loop() {
		timer.Reset(time.Hour)
	}
}
