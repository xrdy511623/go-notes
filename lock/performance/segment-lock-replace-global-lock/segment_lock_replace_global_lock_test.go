package segment_lock_replace_global_lock

import (
	"strconv"
	"sync"
	"testing"
)

func benchmark(b *testing.B, m Map, read, write int) {
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for k := 0; k < read*1000; k++ {
			wg.Add(1)
			go func(k int) {
				m.Set(strconv.Itoa(k), strconv.Itoa(k))
				wg.Done()
			}(k)
		}
		for k := 0; k < write*1000; k++ {
			wg.Add(1)
			go func(k int) {
				m.Get(strconv.Itoa(k))
				wg.Done()
			}(k)
		}
		wg.Wait()
	}
}

var (
	lm = NewLockedMap()
	sm = NewSegmentMap()
)

/*
三种场景，分别使用 全局锁 和 分段锁 测试，共 6 个用例。
每次测试读写操作合计 10000 次，例如读多写少场景，读 9000 次，写 1000 次。
使用 sync.WaitGroup 阻塞直到读写操作全部运行结束。
通过benchmark性能对比测试，可以看到:
读写比为 9:1 时，分段锁的性能比全局锁性能提升28.5%；
读写比为 1:9 时，分段锁和全局锁性能相当；
读写比为 5:5 时，分段锁的性能比全局锁性能提升20.9%；
go test -bench=^Bench -benchtime=5s -benchmem .
goos: darwin
goarch: amd64
pkg: go-notes/lock/performance/segment-lock-replace-global-lock
cpu: Intel(R) Core(TM) i9-9880H CPU @ 2.30GHz
BenchmarkReadMoreLM-16              1302           4905442 ns/op          645666 B/op      38831 allocs/op
BenchmarkReadMoreSM-16              1718           3515601 ns/op          632683 B/op      38701 allocs/op
BenchmarkWriteMoreLM-16             1732           3512546 ns/op          601406 B/op      30708 allocs/op
BenchmarkWriteMoreSM-16             1759           3447555 ns/op          600679 B/op      30701 allocs/op
BenchmarkEqualLM-16                 1417           4349387 ns/op          620102 B/op      34736 allocs/op
BenchmarkEqualSM-16                 1708           3425395 ns/op          616688 B/op      34701 allocs/op
PASS
ok      go-notes/lock/performance/segment-lock-replace-global-lock      39.655s
*/

func BenchmarkReadMoreLM(b *testing.B)  { benchmark(b, lm, 9, 1) }
func BenchmarkReadMoreSM(b *testing.B)  { benchmark(b, sm, 9, 1) }
func BenchmarkWriteMoreLM(b *testing.B) { benchmark(b, lm, 1, 9) }
func BenchmarkWriteMoreSM(b *testing.B) { benchmark(b, sm, 1, 9) }
func BenchmarkEqualLM(b *testing.B)     { benchmark(b, lm, 5, 5) }
func BenchmarkEqualSM(b *testing.B)     { benchmark(b, sm, 5, 5) }
