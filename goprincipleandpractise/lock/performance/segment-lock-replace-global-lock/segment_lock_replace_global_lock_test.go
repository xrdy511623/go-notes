package segmentlockreplacegloballock

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
				m.Get(strconv.Itoa(k))
				wg.Done()
			}(k)
		}
		for k := 0; k < write*1000; k++ {
			wg.Add(1)
			go func(k int) {
				m.Set(strconv.Itoa(k), strconv.Itoa(k))
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
 三种场景的性能对比
  场景: 读多写少 (9:1)
  全局锁 (LM) 均值 ns/op: (1,602,286 + 1,655,972 + 1,647,774 + 1,659,242 + 1,612,849) / 5 = 1,635,625
  分段锁 (SM) 均值 ns/op: (1,605,200 + 1,549,359 + 1,537,348 + 1,535,323 + 1,537,347) / 5 = 1,552,915
  分段锁提升: 5.1%
  ────────────────────────────────────────
  场景: 写多读少 (1:9)
  全局锁 (LM) 均值 ns/op: (1,948,247 + 1,956,416 + 1,941,660 + 2,039,268 + 1,936,816) / 5 = 1,964,481
  分段锁 (SM) 均值 ns/op: (1,555,244 + 1,554,358 + 1,553,923 + 1,663,051 + 1,571,243) / 5 = 1,579,564
  分段锁提升: 19.6%
  ────────────────────────────────────────
  场景: 读写相当 (5:5)
  全局锁 (LM) 均值 ns/op: (1,812,662 + 1,779,725 + 1,777,880 + 1,774,022 + 1,773,699) / 5 = 1,783,598
  分段锁 (SM) 均值 ns/op: (1,597,896 + 1,541,186 + 1,517,925 + 1,525,155 + 1,516,697) / 5 = 1,539,772
  分段锁提升: 13.7%

go test -bench=^Bench -benchtime=10s -count
=5 -benchmem .
goos: darwin
goarch: arm64
pkg: go-notes/goprincipleandpractise/lock/performance/segment-lock-replace-global-lock
cpu: Apple M4
BenchmarkReadMoreLM-10              6663           1602286 ns/op          601302 B/op      30707 allocs/op
BenchmarkReadMoreLM-10              7591           1655972 ns/op          601477 B/op      30708 allocs/op
BenchmarkReadMoreLM-10              7448           1647774 ns/op          601480 B/op      30707 allocs/op
BenchmarkReadMoreLM-10              7513           1659242 ns/op          601210 B/op      30706 allocs/op
BenchmarkReadMoreLM-10              7539           1612849 ns/op          601191 B/op      30706 allocs/op
BenchmarkReadMoreSM-10              7755           1605200 ns/op          600680 B/op      30701 allocs/op
BenchmarkReadMoreSM-10              7922           1549359 ns/op          600679 B/op      30701 allocs/op
BenchmarkReadMoreSM-10              7514           1537348 ns/op          600680 B/op      30701 allocs/op
BenchmarkReadMoreSM-10              7911           1535323 ns/op          600935 B/op      30701 allocs/op
BenchmarkReadMoreSM-10              7022           1537347 ns/op          600674 B/op      30701 allocs/op
BenchmarkWriteMoreLM-10             6062           1948247 ns/op          633663 B/op      38711 allocs/op
BenchmarkWriteMoreLM-10             6202           1956416 ns/op          633721 B/op      38711 allocs/op
BenchmarkWriteMoreLM-10             6140           1941660 ns/op          633685 B/op      38711 allocs/op
BenchmarkWriteMoreLM-10             6220           2039268 ns/op          633779 B/op      38712 allocs/op
BenchmarkWriteMoreLM-10             6235           1936816 ns/op          633673 B/op      38711 allocs/op
BenchmarkWriteMoreSM-10             7760           1555244 ns/op          632680 B/op      38701 allocs/op
BenchmarkWriteMoreSM-10             7263           1554358 ns/op          632684 B/op      38701 allocs/op
BenchmarkWriteMoreSM-10             7128           1553923 ns/op          632682 B/op      38701 allocs/op
BenchmarkWriteMoreSM-10             7767           1663051 ns/op          632683 B/op      38701 allocs/op
BenchmarkWriteMoreSM-10             7179           1571243 ns/op          632685 B/op      38701 allocs/op
BenchmarkEqualLM-10                 6754           1812662 ns/op          617530 B/op      34709 allocs/op
BenchmarkEqualLM-10                 6858           1779725 ns/op          617409 B/op      34708 allocs/op
BenchmarkEqualLM-10                 6846           1777880 ns/op          617439 B/op      34708 allocs/op
BenchmarkEqualLM-10                 6757           1774022 ns/op          617442 B/op      34708 allocs/op
BenchmarkEqualLM-10                 6469           1773699 ns/op          617426 B/op      34708 allocs/op
BenchmarkEqualSM-10                 7978           1597896 ns/op          616684 B/op      34701 allocs/op
BenchmarkEqualSM-10                 7911           1541186 ns/op          616691 B/op      34701 allocs/op
BenchmarkEqualSM-10                 7965           1517925 ns/op          616685 B/op      34701 allocs/op
BenchmarkEqualSM-10                 7986           1525155 ns/op          616684 B/op      34701 allocs/op
BenchmarkEqualSM-10                 7860           1516697 ns/op          616683 B/op      34701 allocs/op
PASS
ok      go-notes/goprincipleandpractise/lock/performance/segment-lock-replace-global-lock       365.147s
*/

func BenchmarkReadMoreLM(b *testing.B)  { benchmark(b, lm, 9, 1) }
func BenchmarkReadMoreSM(b *testing.B)  { benchmark(b, sm, 9, 1) }
func BenchmarkWriteMoreLM(b *testing.B) { benchmark(b, lm, 1, 9) }
func BenchmarkWriteMoreSM(b *testing.B) { benchmark(b, sm, 1, 9) }
func BenchmarkEqualLM(b *testing.B)     { benchmark(b, lm, 5, 5) }
func BenchmarkEqualSM(b *testing.B)     { benchmark(b, sm, 5, 5) }
