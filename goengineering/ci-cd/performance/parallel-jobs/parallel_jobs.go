package paralleljobs

/*
性能对比：串行流水线 vs 并行流水线

CI 中无依赖关系的 job 应该并行执行，显著缩短总时间。

运行基准测试：
  go test -bench=. -benchmem -benchtime=3s .

关键结论：
  - 串行：lint(30s) + test(60s) + security(20s) = 110s
  - 并行：max(lint, test, security) = 60s，节省 45%
  - 并行度受 CI runner 数量限制
*/

import (
	"crypto/sha256"
	"sync"
)

// PipelineStage 模拟一个流水线阶段
type PipelineStage struct {
	Name       string
	Iterations int // 模拟工作量
}

// StandardPipeline 返回标准的 Go CI 流水线阶段
func StandardPipeline() []PipelineStage {
	return []PipelineStage{
		{Name: "lint", Iterations: 300},
		{Name: "unit-test", Iterations: 600},
		{Name: "security-scan", Iterations: 200},
		{Name: "vet", Iterations: 150},
	}
}

// executeStage 模拟执行一个流水线阶段
func executeStage(stage PipelineStage) [32]byte {
	data := []byte(stage.Name)
	hash := sha256.Sum256(data)
	for i := 0; i < stage.Iterations; i++ {
		hash = sha256.Sum256(hash[:])
	}
	return hash
}

// SequentialPipeline 串行执行所有阶段
func SequentialPipeline(stages []PipelineStage) [][32]byte {
	results := make([][32]byte, len(stages))
	for i, stage := range stages {
		results[i] = executeStage(stage)
	}
	return results
}

// ParallelPipeline 并行执行所有阶段
func ParallelPipeline(stages []PipelineStage) [][32]byte {
	results := make([][32]byte, len(stages))
	var wg sync.WaitGroup

	for i, stage := range stages {
		wg.Add(1)
		go func(idx int, s PipelineStage) {
			defer wg.Done()
			results[idx] = executeStage(s)
		}(i, stage)
	}

	wg.Wait()
	return results
}
