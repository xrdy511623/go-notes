package segmentlockreplacegloballock

import (
	"strconv"
	"sync/atomic"
	"testing"
)

const keySpace = 4096

/*
对比全局锁 map 与分段锁 map 在不同读写比例下的并发开销。

执行命令:

	go test -run '^$' -bench '^Benchmark' -benchtime=3s -count=5 -benchmem .

Apple M4(Go 1.24.5)下5次均值:

	读多(9:1): LM 97.680 ns/op, SM 43.156 ns/op, SM快 2.26x
	写多(1:9): LM 98.118 ns/op, SM 55.146 ns/op, SM快 1.78x
	均衡(5:5): LM 64.938 ns/op, SM 50.182 ns/op, SM快 1.29x
*/
func benchmarkMixed(b *testing.B, newMap func() Map, readWeight, writeWeight uint64) {
	total := readWeight + writeWeight
	if total == 0 {
		b.Fatal("read/write weights must not both be zero")
	}

	keys := make([]string, keySpace)
	for i := range keys {
		keys[i] = strconv.Itoa(i)
	}

	m := newMap()
	for _, key := range keys {
		m.Set(key, key)
	}

	var seq atomic.Uint64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			n := seq.Add(1) - 1
			key := keys[int(n%uint64(len(keys)))]
			if n%total < readWeight {
				m.Get(key)
				continue
			}
			m.Set(key, key)
		}
	})
}

func BenchmarkReadMoreLM(b *testing.B) {
	benchmarkMixed(b, NewLockedMap, 9, 1)
}

func BenchmarkReadMoreSM(b *testing.B) {
	benchmarkMixed(b, func() Map { return NewSegmentMap() }, 9, 1)
}

func BenchmarkWriteMoreLM(b *testing.B) {
	benchmarkMixed(b, NewLockedMap, 1, 9)
}

func BenchmarkWriteMoreSM(b *testing.B) {
	benchmarkMixed(b, func() Map { return NewSegmentMap() }, 1, 9)
}

func BenchmarkEqualLM(b *testing.B) {
	benchmarkMixed(b, NewLockedMap, 5, 5)
}

func BenchmarkEqualSM(b *testing.B) {
	benchmarkMixed(b, func() Map { return NewSegmentMap() }, 5, 5)
}
