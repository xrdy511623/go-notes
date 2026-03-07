package parallelbuild

import (
	"fmt"
	"runtime"
	"testing"
)

/*
基准测试：串行构建 vs 并行构建

运行方式：
  go test -bench=. -benchmem -benchtime=3s .

预期结果（8 核机器，16 个编译单元）：
  BenchmarkSequentialBuild-8     xxx    yyy ns/op    zzz B/op    n allocs/op
  BenchmarkParallelBuild_2-8     xxx    yyy ns/op    ...
  BenchmarkParallelBuild_4-8     xxx    yyy ns/op    ...
  BenchmarkParallelBuild_8-8     xxx    yyy ns/op    ...

  并行度     相对串行的加速比
  2 workers   ~1.8x
  4 workers   ~3.5x
  8 workers   ~6x

结论：
  - 并行构建的加速比接近但低于理论值（线程调度开销）
  - 超过 CPU 核数后加速比趋于平缓
  - go build 默认使用 GOMAXPROCS 并行度，通常是最优选择
*/

func BenchmarkSequentialBuild(b *testing.B) {
	units := NewCompileUnits(16)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SequentialBuild(units)
	}
}

func BenchmarkParallelBuild(b *testing.B) {
	units := NewCompileUnits(16)
	workerCounts := []int{2, 4, 8, runtime.NumCPU()}

	for _, workers := range workerCounts {
		b.Run(fmt.Sprintf("workers=%d", workers), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				ParallelBuild(units, workers)
			}
		})
	}
}
