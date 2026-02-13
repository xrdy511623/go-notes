package performance

import (
	"testing"
)

/*
strings.Clone vs 子串切片内存保留对比

执行命令:

	go test -run '^$' -bench 'Substring' -benchmem .

对比维度:
  1. 直接切片: s[:n] — 零分配，但共享底层大数组，导致内存保留
  2. strings.Clone: 独立拷贝，分配新内存，允许 GC 回收原始大字符串
  3. []byte 拷贝: Go 1.20 之前的手动拷贝方式

结论:
  - 直接切片速度最快（零分配），但会阻止大字符串被 GC
  - strings.Clone 仅分配子串大小的内存，允许原始大字符串被回收
  - 当从大字符串中提取小子串并长期持有时，务必使用 strings.Clone
  - []byte 拷贝与 Clone 性能接近，但 Clone 语义更清晰
*/

const (
	largeSize  = 1 << 20 // 1 MB
	substrSize = 32
)

func BenchmarkSubstringDirect(b *testing.B) {
	s := GenerateLargeString(largeSize)
	b.ResetTimer()
	for b.Loop() {
		cloneSink = SubstringDirect(s, substrSize)
	}
}

func BenchmarkSubstringClone(b *testing.B) {
	s := GenerateLargeString(largeSize)
	b.ResetTimer()
	for b.Loop() {
		cloneSink = SubstringClone(s, substrSize)
	}
}

func BenchmarkSubstringConcatCopy(b *testing.B) {
	s := GenerateLargeString(largeSize)
	b.ResetTimer()
	for b.Loop() {
		cloneSink = SubstringConcatCopy(s, substrSize)
	}
}
