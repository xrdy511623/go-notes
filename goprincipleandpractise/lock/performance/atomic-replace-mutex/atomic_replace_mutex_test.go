package atomicreplacemutex

import (
	"sync/atomic"
	"testing"
)

/*
对比无同步、atomic、mutex三种并发累加方式的开销。

执行命令:

	go test -run '^$' -bench '^Benchmark' -benchtime=3s -count=5 -benchmem .

Apple M4(Go 1.24.5)下5次均值:

	Baseline(local++)   0.057 ns/op
	Atomic(AddInt64)   31.930 ns/op
	Mutex(Lock/Unlock) 62.298 ns/op

结论:
 1. 原子操作约比互斥锁快 1.95x（约降低 48.7% 延迟）。
 2. baseline 只是无同步下限，不具备并发安全语义，不能用于业务实现。
*/
func BenchmarkAddNormal(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		var local int64
		for pb.Next() {
			local++
		}
		_ = local
	})
}

func BenchmarkAddUseAtomic(b *testing.B) {
	c := new(counterAtomic)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			atomic.AddInt64(&c.i, 1)
		}
	})
}

func BenchmarkAddUseMutex(b *testing.B) {
	c := new(counterMutex)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.m.Lock()
			c.i++
			c.m.Unlock()
		}
	})
}
