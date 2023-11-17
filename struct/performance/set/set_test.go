package set

import "testing"

var s1 = make(boolSet)
var s2 = make(emptyStructSet)

/*
从下面的测试结果可以看出，使用空结构体作为Set(集合)的值，比使用布尔值性能还是稍有优势的。
此案例中，提升性能大约0.58%

go test -bench=Set$ -benchmem .
goos: darwin
goarch: arm64
pkg: go-notes/struct/performance/set
BenchmarkBoolSet-8                  1778            668528 ns/op          116642 B/op      29700 allocs/op
BenchmarkEmptyStructSet-8           1796            664681 ns/op          116640 B/op      29700 allocs/op
PASS
ok      go-notes/struct/performance/set 2.960s

*/

func BenchmarkBoolSet(b *testing.B)        { Benchmark(b, 10000, s1) }
func BenchmarkEmptyStructSet(b *testing.B) { Benchmark(b, 10000, s2) }
