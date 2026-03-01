package parallelstages

import (
	"fmt"
	"testing"
)

/*
基准测试：串行 Stage vs 并行 Stage

运行方式：
  go test -bench=. -benchmem -benchtime=3s .

预期结果（多核机器）：
  BenchmarkSequentialStages-8    xxx    yyy ns/op    zzz B/op    n allocs/op
  BenchmarkParallelStages-8     xxx    yyy ns/op    zzz B/op    n allocs/op

  模式           模拟耗时      实际比例
  串行           10min        sum(3+5+2)
  并行           5min         max(3,5,2)
  节省           50%

结论：
  - 并行 Stage 总时间由最慢的 Stage 决定
  - Go CI 典型场景（Lint + Test + Security）并行可节省 40-60%
  - 实际节省取决于 Jenkins Agent 是否有足够的 executor
  - 并行 Stage 之间不能有依赖关系（如 Build 依赖 Lint 通过）
*/

func BenchmarkSequentialStages(b *testing.B) {
	stages := NewCIStages()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SequentialStages(stages)
	}
}

func BenchmarkParallelStages(b *testing.B) {
	stages := NewCIStages()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParallelStages(stages)
	}
}

func BenchmarkParallelStages_VaryingCount(b *testing.B) {
	stageCounts := []int{2, 3, 5, 8}
	for _, count := range stageCounts {
		stages := make([]Stage, count)
		for i := range stages {
			stages[i] = Stage{
				Name: fmt.Sprintf("Stage%d", i),
				Work: cpuWork(200 + i*100),
			}
		}
		b.Run(fmt.Sprintf("stages=%d", count), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				ParallelStages(stages)
			}
		})
	}
}
