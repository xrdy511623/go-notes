package timenowcost

import (
	"testing"
	"time"
)

/*
测量 time.Now() 及相关操作的调用开销。

执行命令:

	go test -run '^$' -bench '^Benchmark' -benchtime=3s -count=5 -benchmem .

说明:
  time.Now() 底层是一个系统调用 (vDSO 优化后通常很快)。
  这组 benchmark 量化其开销，帮助判断在热路径上频繁调用 time.Now() 是否可接受。
*/

var (
	sinkTime     time.Time
	sinkDuration time.Duration
	sinkInt64    int64
)

// BenchmarkTimeNow measures the cost of a single time.Now() call.
func BenchmarkTimeNow(b *testing.B) {
	for b.Loop() {
		sinkTime = time.Now()
	}
}

// BenchmarkTimeSince measures time.Since (time.Now().Sub(start)).
func BenchmarkTimeSince(b *testing.B) {
	start := time.Now()
	b.ResetTimer()
	for b.Loop() {
		sinkDuration = time.Since(start)
	}
}

// BenchmarkTimeUnixNano measures time.Now().UnixNano().
func BenchmarkTimeUnixNano(b *testing.B) {
	for b.Loop() {
		sinkInt64 = time.Now().UnixNano()
	}
}

// BenchmarkTimeSub measures Sub between two pre-computed times.
func BenchmarkTimeSub(b *testing.B) {
	t1 := time.Now()
	time.Sleep(time.Millisecond)
	t2 := time.Now()
	b.ResetTimer()
	for b.Loop() {
		sinkDuration = t2.Sub(t1)
	}
}

// BenchmarkTimeEqual measures the Equal comparison.
func BenchmarkTimeEqual(b *testing.B) {
	t1 := time.Now()
	t2 := t1.In(time.UTC)
	b.ResetTimer()
	var sink bool
	for b.Loop() {
		sink = t1.Equal(t2)
	}
	_ = sink
}

// BenchmarkTimeFormat measures Format as a reference point.
func BenchmarkTimeFormat(b *testing.B) {
	t := time.Now()
	b.ResetTimer()
	var sink string
	for b.Loop() {
		sink = t.Format(time.RFC3339)
	}
	_ = sink
}
