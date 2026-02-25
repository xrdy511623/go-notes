package poolsizetuning

import (
	"fmt"
	"sync/atomic"
	"testing"
)

/*
对比不同连接池大小对并发吞吐量的影响。

执行命令:

	go test -run '^$' -bench '^Benchmark' -benchtime=3s -count=5 -benchmem .
*/

func benchmarkPoolSize(b *testing.B, maxOpen int) {
	db, err := NewDB(maxOpen, maxOpen)
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	var seq atomic.Int64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			id := int(seq.Add(1))
			if err := DoWork(db, id); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkPoolSize1(b *testing.B)  { benchmarkPoolSize(b, 1) }
func BenchmarkPoolSize5(b *testing.B)  { benchmarkPoolSize(b, 5) }
func BenchmarkPoolSize10(b *testing.B) { benchmarkPoolSize(b, 10) }
func BenchmarkPoolSize25(b *testing.B) { benchmarkPoolSize(b, 25) }

func BenchmarkPoolSizeReport(b *testing.B) {
	sizes := []int{1, 5, 10, 25}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("MaxOpen=%d", size), func(b *testing.B) {
			benchmarkPoolSize(b, size)
		})
	}
}
