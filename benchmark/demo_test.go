package benchmark_test

import (
	bench "go-notes/benchmark"
	"testing"
)

/*
以下求斐波拉契数的四种算法时间复杂度可以很直观的看出来，如果不能直观的看出这种性能差异，可以使用benchmark做性能基准测试
*/

func BenchmarkFib(b *testing.B)                      { bench.Fib(50) }
func BenchmarkFibUseCache(b *testing.B)              { bench.FibUseCache(50) }
func BenchmarkFibUseDynamicProgramming(b *testing.B) { bench.FibUseDynamicProgramming(50) }
func BenchmarkFibSimple(b *testing.B)                { bench.FibSimple(50) }
