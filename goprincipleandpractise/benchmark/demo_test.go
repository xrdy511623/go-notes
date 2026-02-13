package benchmark

import "testing"

/*
以下求斐波拉契数的四种算法时间复杂度可以很直观的看出来，如果不能直观的看出这种性能差异，可以使用benchmark做性能基准测试
*/

var fibSink int

func BenchmarkFib(b *testing.B) {
	for b.Loop() {
		fibSink = Fib(50)
	}
}

func BenchmarkFibUseCache(b *testing.B) {
	for b.Loop() {
		fibSink = FibUseCache(50)
	}
}

func BenchmarkFibUseDynamicProgramming(b *testing.B) {
	for b.Loop() {
		fibSink = FibUseDynamicProgramming(50)
	}
}

func BenchmarkFibSimple(b *testing.B) {
	for b.Loop() {
		fibSink = FibSimple(50)
	}
}
