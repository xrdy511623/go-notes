package narrowcriticalspace

import "testing"

var c = new(counter)

/*
缩小临界区，有助于提升性能
 go test -bench=^Bench -benchtime=10s -count=5 -benchmem .
goos: darwin
goarch: arm64
pkg: go-notes/goprincipleandpractise/lock/performance/narrow-critical-space
cpu: Apple M4
BenchmarkCountDefer-10            724573             15525 ns/op               0 B/op          0 allocs/op
BenchmarkCountDefer-10            750994             15398 ns/op               0 B/op          0 allocs/op
BenchmarkCountDefer-10            742126             15675 ns/op               0 B/op          0 allocs/op
BenchmarkCountDefer-10            755752             15365 ns/op               0 B/op          0 allocs/op
BenchmarkCountDefer-10            739354             15563 ns/op               0 B/op          0 allocs/op
BenchmarkCountNarrow-10           746234             15693 ns/op               0 B/op          0 allocs/op
BenchmarkCountNarrow-10           736068             14579 ns/op               0 B/op          0 allocs/op
BenchmarkCountNarrow-10           724306             14363 ns/op               0 B/op          0 allocs/op
BenchmarkCountNarrow-10           731962             14873 ns/op               0 B/op          0 allocs/op
BenchmarkCountNarrow-10           731373             15408 ns/op               0 B/op          0 allocs/op
PASS
ok      go-notes/goprincipleandpractise/lock/performance/narrow-critical-space  114.408s
*/

func BenchmarkCountDefer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		countDefer(c)
	}
}

func BenchmarkCountNarrow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		countNarrow(c)
	}
}
