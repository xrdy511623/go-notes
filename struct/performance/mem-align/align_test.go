package mem_align

import "testing"

/*
可以看到，经过三轮benchmark对比测试的结果，两个结构体，字段数和字段的类型完全相同，但稍微调整下字段的顺序，却可以
起到提高程序性能的效果，这个案例中，合理安排结构体字段顺序将性能提升了大约39%。
go test -bench=Struct$ -benchmem -count=3 .
goos: darwin
goarch: arm64
pkg: go-notes/struct/performance/mem-align
BenchmarkUseOrderStruct-8       1000000000               0.003623 ns/op        0 B/op          0 allocs/op
BenchmarkUseOrderStruct-8       1000000000               0.003415 ns/op        0 B/op          0 allocs/op
BenchmarkUseOrderStruct-8       1000000000               0.003182 ns/op        0 B/op          0 allocs/op
BenchmarkUseDisOrderStruct-8    1000000000               0.006061 ns/op        0 B/op          0 allocs/op
BenchmarkUseDisOrderStruct-8    1000000000               0.005135 ns/op        0 B/op          0 allocs/op
BenchmarkUseDisOrderStruct-8    1000000000               0.005610 ns/op        0 B/op          0 allocs/op
PASS
ok      go-notes/struct/performance/mem-align   0.285s

*/

func BenchmarkUseOrderStruct(b *testing.B) {
	UseOrderStruct(1000000)
}

func BenchmarkUseDisOrderStruct(b *testing.B) {
	UseDisOrderStruct(1000000)
}
