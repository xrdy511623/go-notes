package rw_lock_replace_mutex

import (
	"sync"
	"testing"
)

func benchmark(b *testing.B, rw RW, read, write int) {
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for k := 0; k < read*1000; k++ {
			wg.Add(1)
			go func() {
				rw.Read()
				wg.Done()
			}()
		}
		for k := 0; k < write*1000; k++ {
			wg.Add(1)
			go func() {
				rw.Write()
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

/*
三种场景，分别使用 Lock 和 RWLock 测试，共 6 个用例。
每次测试读写操作合计 10000 次，例如读多写少场景，读 9000 次，写 1000 次。
使用 sync.WaitGroup 阻塞直到读写操作全部运行结束。
通过benchmark性能对比测试，可以看到:
读写比为 9:1 时，读写锁的性能约为互斥锁的 6.5 倍
读写比为 1:9 时，读写锁性能相当
读写比为 5:5 时，读写锁的性能约为互斥锁的 2 倍
go test -bench=^Bench -benchtime=5s -benchmem .
goos: darwin
goarch: amd64
pkg: go-notes/lock/performance/rw-lock-replace-mutex
cpu: Intel(R) Core(TM) i9-9880H CPU @ 2.30GHz
BenchmarkReadMore-16                 100          53971224 ns/op         1246504 B/op      21219 allocs/op
BenchmarkReadMoreRW-16               655           9061475 ns/op         1133192 B/op      20138 allocs/op
BenchmarkWriteMore-16                100          53521364 ns/op         1154239 B/op      20357 allocs/op
BenchmarkWriteMoreRW-16              122          52892140 ns/op         1166760 B/op      20487 allocs/op
BenchmarkEqual-16                     94          55132050 ns/op         1136588 B/op      20173 allocs/op
BenchmarkEqualRW-16                  201          29987758 ns/op         1218998 B/op      21032 allocs/op
PASS
ok      go-notes/lock/performance/rw-lock-replace-mutex 44.083s
*/

func BenchmarkReadMore(b *testing.B)    { benchmark(b, &Lock{}, 9, 1) }
func BenchmarkReadMoreRW(b *testing.B)  { benchmark(b, &RWLock{}, 9, 1) }
func BenchmarkWriteMore(b *testing.B)   { benchmark(b, &Lock{}, 1, 9) }
func BenchmarkWriteMoreRW(b *testing.B) { benchmark(b, &RWLock{}, 1, 9) }
func BenchmarkEqual(b *testing.B)       { benchmark(b, &Lock{}, 5, 5) }
func BenchmarkEqualRW(b *testing.B)     { benchmark(b, &RWLock{}, 5, 5) }
