package rwlockreplacemutex

import (
	"sync/atomic"
	"testing"
)

/*
对比 Mutex 与 RWMutex 在不同读写比例下的并发开销。

执行命令:

	go test -run '^$' -bench '^Benchmark' -benchtime=3s -count=5 -benchmem .

Apple M4(Go 1.24.5)下5次均值:

	读多(9:1): Mutex 3712.0 ns/op, RWMutex 1047.8 ns/op, RWMutex快 3.54x
	写多(1:9): Mutex 3690.2 ns/op, RWMutex 3687.4 ns/op, 性能基本持平
	均衡(5:5): Mutex 3673.0 ns/op, RWMutex 3281.8 ns/op, RWMutex快 1.12x
*/
func benchmarkMixed(b *testing.B, rw RW, readWeight, writeWeight uint64) {
	total := readWeight + writeWeight
	if total == 0 {
		b.Fatal("read/write weights must not both be zero")
	}

	var seq atomic.Uint64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			n := seq.Add(1) - 1
			if n%total < readWeight {
				rw.Read()
				continue
			}
			rw.Write()
		}
	})
}

func BenchmarkReadMore(b *testing.B)    { benchmarkMixed(b, &Lock{}, 9, 1) }
func BenchmarkReadMoreRW(b *testing.B)  { benchmarkMixed(b, &RWLock{}, 9, 1) }
func BenchmarkWriteMore(b *testing.B)   { benchmarkMixed(b, &Lock{}, 1, 9) }
func BenchmarkWriteMoreRW(b *testing.B) { benchmarkMixed(b, &RWLock{}, 1, 9) }
func BenchmarkEqual(b *testing.B)       { benchmarkMixed(b, &Lock{}, 5, 5) }
func BenchmarkEqualRW(b *testing.B)     { benchmarkMixed(b, &RWLock{}, 5, 5) }
