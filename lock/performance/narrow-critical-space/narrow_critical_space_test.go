package narrow_critical_space

import "testing"

var c = new(counter)

/*
缩小临界区，有助于提升性能
go test -bench=^Bench -benchtime=5s -benchmem .
goos: darwin
goarch: amd64
pkg: go-notes/lock/performance/narrow-critical-space
cpu: Intel(R) Core(TM) i9-9880H CPU @ 2.30GHz
BenchmarkCountDefer-16          1000000000               0.0000145 ns/op               0 B/op          0 allocs/op
BenchmarkCountNarrow-16         1000000000               0.0000114 ns/op               0 B/op          0 allocs/op
PASS
ok      go-notes/lock/performance/narrow-critical-space 0.445s
*/

func BenchmarkCountDefer(b *testing.B) {
	countDefer(c)
}

func BenchmarkCountNarrow(b *testing.B) {
	countNarrow(c)
}
