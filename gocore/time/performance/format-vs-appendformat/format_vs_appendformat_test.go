package formatvsappendformat

import (
	"testing"
	"time"
)

/*
对比 time.Format vs time.AppendFormat 的性能差异。

执行命令:

	go test -run '^$' -bench '^Benchmark' -benchtime=3s -count=5 -benchmem .

预期结果:
  - Format 每次分配一个新 string
  - AppendFormat 复用 buffer，allocs/op 为 0
  - 在高频格式化场景（日志、序列化）差异显著
*/

var (
	benchTime = time.Date(2024, 6, 15, 14, 30, 0, 123456789, time.UTC)
	sink      string
	sinkBytes []byte
)

func BenchmarkFormatRFC3339(b *testing.B) {
	for b.Loop() {
		sink = FormatRFC3339(benchTime)
	}
}

func BenchmarkAppendFormatRFC3339(b *testing.B) {
	buf := make([]byte, 0, 64)
	b.ResetTimer()
	for b.Loop() {
		sinkBytes = AppendFormatRFC3339(buf[:0], benchTime)
	}
}

func BenchmarkFormatCustom(b *testing.B) {
	for b.Loop() {
		sink = FormatCustom(benchTime)
	}
}

func BenchmarkAppendFormatCustom(b *testing.B) {
	buf := make([]byte, 0, 64)
	b.ResetTimer()
	for b.Loop() {
		sinkBytes = AppendFormatCustom(buf[:0], benchTime)
	}
}

// BenchmarkFormatBatch simulates formatting many times (e.g., logging).
func BenchmarkFormatBatch1000(b *testing.B) {
	for b.Loop() {
		for j := 0; j < 1000; j++ {
			sink = benchTime.Format(time.RFC3339)
		}
	}
}

func BenchmarkAppendFormatBatch1000(b *testing.B) {
	buf := make([]byte, 0, 64)
	b.ResetTimer()
	for b.Loop() {
		for j := 0; j < 1000; j++ {
			sinkBytes = benchTime.AppendFormat(buf[:0], time.RFC3339)
		}
	}
}
