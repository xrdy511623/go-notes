package paralleljobs

import "testing"

/*
基准测试：串行 vs 并行流水线

运行方式：
  go test -bench=. -benchmem -benchtime=3s .

预期结果：
  BenchmarkSequentialPipeline-8    xxx    yyy ns/op
  BenchmarkParallelPipeline-8      xxx    yyy ns/op

  并行执行在多核机器上显著更快，
  加速比接近 stages 数量（受限于最慢的 stage）。
*/

func BenchmarkSequentialPipeline(b *testing.B) {
	stages := StandardPipeline()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SequentialPipeline(stages)
	}
}

func BenchmarkParallelPipeline(b *testing.B) {
	stages := StandardPipeline()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParallelPipeline(stages)
	}
}
