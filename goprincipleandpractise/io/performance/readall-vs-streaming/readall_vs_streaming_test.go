package readallvsstreaming

import (
	"testing"
)

/*
基准测试：io.ReadAll（全量加载） vs 流式读取（固定缓冲区）

运行：
  go test -run '^$' -bench '^Benchmark' -benchtime=3s -count=3 -benchmem .

预期结论：
  - 10MB：时间差距不大，但 ReadAll 分配远多于 Streaming
  - 100MB：ReadAll 触发 GC 压力，明显变慢
  - Streaming 使用 O(bufSize) 常量内存，适合大数据量处理
  - -benchmem 输出会清楚展示内存分配差异
*/

var pattern = []byte("ABCDEFGHIJKLMNOP") // 16 bytes pattern

func BenchmarkReadAll1MB(b *testing.B) {
	for b.Loop() {
		r := NewRepeatingReader(pattern, 1*1024*1024)
		ProcessReadAll(r)
	}
}

func BenchmarkStreaming1MB(b *testing.B) {
	for b.Loop() {
		r := NewRepeatingReader(pattern, 1*1024*1024)
		ProcessStreaming(r, 32*1024)
	}
}

func BenchmarkReadAll10MB(b *testing.B) {
	for b.Loop() {
		r := NewRepeatingReader(pattern, 10*1024*1024)
		ProcessReadAll(r)
	}
}

func BenchmarkStreaming10MB(b *testing.B) {
	for b.Loop() {
		r := NewRepeatingReader(pattern, 10*1024*1024)
		ProcessStreaming(r, 32*1024)
	}
}

func BenchmarkReadAll50MB(b *testing.B) {
	for b.Loop() {
		r := NewRepeatingReader(pattern, 50*1024*1024)
		ProcessReadAll(r)
	}
}

func BenchmarkStreaming50MB(b *testing.B) {
	for b.Loop() {
		r := NewRepeatingReader(pattern, 50*1024*1024)
		ProcessStreaming(r, 32*1024)
	}
}
