package rwlockreplacemutex

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
读写比为 9:1 时，读写锁的性能约为互斥锁的 7 倍
读写比为 1:9 时，读写锁性能相当
读写比为 5:5 时，读写锁的性能约为互斥锁的 2 倍
 go test -bench=^Bench -benchtime=5s -benchmem .
goos: darwin
goarch: arm64
pkg: go-notes/goprincipleandpractise/lock/performance/rw-lock-replace-mutex
cpu: Apple M4
BenchmarkReadMore-10                 141          42184281 ns/op         1376758 B/op      21008 allocs/op
BenchmarkReadMoreRW-10              1005           6096832 ns/op         1297193 B/op      20179 allocs/op
BenchmarkWriteMore-10                141          42087855 ns/op         1352776 B/op      20758 allocs/op
BenchmarkWriteMoreRW-10              152          38916778 ns/op         1362408 B/op      20859 allocs/op
BenchmarkEqual-10                    140          42788979 ns/op         1378384 B/op      21025 allocs/op
BenchmarkEqualRW-10                  272          22311830 ns/op         1366386 B/op      20900 allocs/op
PASS
ok      go-notes/goprincipleandpractise/lock/performance/rw-lock-replace-mutex  55.825s
*/

func BenchmarkReadMore(b *testing.B)    { benchmark(b, &Lock{}, 9, 1) }
func BenchmarkReadMoreRW(b *testing.B)  { benchmark(b, &RWLock{}, 9, 1) }
func BenchmarkWriteMore(b *testing.B)   { benchmark(b, &Lock{}, 1, 9) }
func BenchmarkWriteMoreRW(b *testing.B) { benchmark(b, &RWLock{}, 1, 9) }
func BenchmarkEqual(b *testing.B)       { benchmark(b, &Lock{}, 5, 5) }
func BenchmarkEqualRW(b *testing.B)     { benchmark(b, &RWLock{}, 5, 5) }
