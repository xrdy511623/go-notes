package atomic_replace_mutex

import (
	"testing"
)

var (
	c1 = new(counter)
	c2 = new(counterAtomic)
	c3 = new(counterMutex)
)

/*
可以看到，使用atomic代替mutex互斥锁，性能可以提升3倍以上。
go test -bench=^Bench -benchtime=5s -benchmem .
goos: darwin
goarch: amd64
pkg: go-notes/lock/performance/atomic-replace-mutex
cpu: Intel(R) Core(TM) i9-9880H CPU @ 2.30GHz
BenchmarkAddNormal-16           1000000000               0.0000018 ns/op               0 B/op          0 allocs/op
BenchmarkAddUseAtomic-16        1000000000               0.0000057 ns/op               0 B/op          0 allocs/op
BenchmarkAddUseMutex-16         1000000000               0.0000173 ns/op               0 B/op          0 allocs/op
PASS
ok      go-notes/lock/performance/atomic-replace-mutex  0.356s

*/

func BenchmarkAddNormal(b *testing.B) {
	add(c1, 1000)
}

func BenchmarkAddUseAtomic(b *testing.B) {
	addUseAtomic(c2, 1000)
}

func BenchmarkAddUseMutex(b *testing.B) {
	addUseMutex(c3, 1000)
}
