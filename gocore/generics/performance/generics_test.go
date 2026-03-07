package performance

import "testing"

/*
从benchmark的对比测试来看，使用泛型与常规类型的函数大致相当，然而，相较于使用反射实现的函数，它的性能高出30多倍。
go test -benchmem . -bench="^Bench"
goos: darwin
goarch: amd64
pkg: go-notes/goprincipleandpractise/generics/performance
cpu: Intel(R) Core(TM) i9-9880H CPU @ 2.30GHz
BenchmarkRegular-16             1000000000               0.2822 ns/op          0 B/op          0 allocs/op
BenchmarkReflection-16          142170075                8.185 ns/op           0 B/op          0 allocs/op
BenchmarkGenerics-16            1000000000               0.2350 ns/op          0 B/op          0 allocs/op
PASS
ok      go-notes/goprincipleandpractise/generics/performance 4.141s
*/

// MaxInt函数benchmark
func BenchmarkRegular(b *testing.B) {
	for i := 0; i < b.N; i++ {
		MaxInt(1, 2)
	}
}

// 反射实现的最大值函数benchmark
func BenchmarkReflection(b *testing.B) {
	for i := 0; i < b.N; i++ {
		MaxUseReflection(1, 2)
	}
}

// 泛型实现的最大值函数benchmark
func BenchmarkGenerics(b *testing.B) {
	for i := 0; i < b.N; i++ {
		MaxUseGenerics(1, 2)
	}
}
