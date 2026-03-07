package falsesharing

import (
	"testing"
	"unsafe"
)

/*
False Sharing（伪共享）性能对比

执行命令:

	go test -run '^$' -bench '^Benchmark' -benchtime=3s -count=3 -benchmem .

对比维度:
  - NoPadCounter: 两个 atomic.Int64 紧挨着（16 字节），在同一 cache line
  - PaddedCounter: 两个 atomic.Int64 之间插入 56 字节 padding，各占独立 cache line

结论:
  - 当两个 goroutine 在不同核心上频繁修改同一 cache line 的不同变量时，
    cache coherence 协议（如 MESI）会不断使对方缓存失效
  - 插入 padding 使变量各占独立 cache line 后，性能提升显著（通常 2-5x）
  - 适用场景：高并发下多 goroutine 频繁修改的共享结构体
  - 代价：每个 padding 浪费约 56 字节内存，需要权衡
*/

func TestStructSize(t *testing.T) {
	t.Logf("NoPadCounter  size=%d", unsafe.Sizeof(NoPadCounter{}))
	t.Logf("PaddedCounter size=%d", unsafe.Sizeof(PaddedCounter{}))
}

func BenchmarkFalseSharingNoPad(b *testing.B) {
	c := &NoPadCounter{}
	for b.Loop() {
		IncrementNoPad(c, 10000)
	}
}

func BenchmarkFalseSharingPadded(b *testing.B) {
	c := &PaddedCounter{}
	for b.Loop() {
		IncrementPadded(c, 10000)
	}
}
