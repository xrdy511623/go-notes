package narrowcriticalspace

import "testing"

/*
对比 defer 解锁与缩小临界区两种写法在并发竞争下的差异。

执行命令:

	go test -run '^$' -bench '^Benchmark' -benchtime=3s -count=5 -benchmem .

Apple M4(Go 1.24.5)下5次均值:

	CountDefer  15934 ns/op
	CountNarrow  1645 ns/op

结论:

	缩小临界区约快 9.69x（约降低 89.7% 延迟）。
*/
func benchmarkCounter(b *testing.B, fn func(*counter)) {
	c := new(counter)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			fn(c)
		}
	})
}

func BenchmarkCountDefer(b *testing.B) {
	benchmarkCounter(b, countDefer)
}

func BenchmarkCountNarrow(b *testing.B) {
	benchmarkCounter(b, countNarrow)
}
