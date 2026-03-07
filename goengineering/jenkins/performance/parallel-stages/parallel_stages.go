package parallelstages

/*
性能对比：Jenkins Pipeline 串行 Stage vs 并行 Stage

在 Jenkins Pipeline 中，无依赖关系的 Stage（如 Lint、Test、Security）
可以用 parallel 块并行执行，显著缩短总流水线时间。

本实验模拟：
  1. 串行执行：Lint(3min) → Test(5min) → Security(2min) = 总计 10min
  2. 并行执行：max(Lint, Test, Security) = 总计 5min

运行基准测试：
  go test -bench=. -benchmem -benchtime=3s ./...

关键结论：
  - 并行 Stage 总时间 = max(各 Stage 时间)，而非 sum
  - 典型 Go CI（Lint + Test + Security 并行）可节省 40-60% 时间
  - 并行 Stage 的瓶颈是最慢的那个 Stage
  - 需要足够的 Agent/executor 资源来支撑并行度

Jenkinsfile 并行配置示例：

  // 串行（慢）
  stage('Lint')     { steps { sh 'golangci-lint run ./...' } }
  stage('Test')     { steps { sh 'go test -race ./...' } }
  stage('Security') { steps { sh 'gosec ./...' } }

  // 并行（快）
  stage('Quality') {
      parallel {
          stage('Lint')     { steps { sh 'golangci-lint run ./...' } }
          stage('Test')     { steps { sh 'go test -race ./...' } }
          stage('Security') { steps { sh 'gosec ./...' } }
      }
  }
*/

import (
	"crypto/sha256"
	"sync"
	"time"
)

// Stage 模拟一个 Pipeline Stage
type Stage struct {
	Name     string
	Duration time.Duration // 模拟执行时间
	Work     func()        // 模拟的工作负载
}

// NewCIStages 创建一组典型 Go CI Stage
func NewCIStages() []Stage {
	return []Stage{
		{
			Name:     "Lint",
			Duration: 3 * time.Second, // 模拟 3 分钟 → 缩放为 3 秒
			Work:     cpuWork(300),
		},
		{
			Name:     "Test",
			Duration: 5 * time.Second, // 模拟 5 分钟（最慢）
			Work:     cpuWork(500),
		},
		{
			Name:     "Security",
			Duration: 2 * time.Second, // 模拟 2 分钟
			Work:     cpuWork(200),
		},
	}
}

// cpuWork 生成指定轮数的 CPU 密集工作
func cpuWork(rounds int) func() {
	return func() {
		data := []byte("jenkins-pipeline-stage-simulation")
		hash := sha256.Sum256(data)
		for i := 0; i < rounds; i++ {
			hash = sha256.Sum256(hash[:])
		}
	}
}

// SequentialStages 串行执行所有 Stage
// 总时间 = sum(各 Stage 时间)
func SequentialStages(stages []Stage) time.Duration {
	start := time.Now()
	for _, s := range stages {
		s.Work()
	}
	return time.Since(start)
}

// ParallelStages 并行执行所有 Stage
// 总时间 ≈ max(各 Stage 时间)
func ParallelStages(stages []Stage) time.Duration {
	start := time.Now()
	var wg sync.WaitGroup
	for _, s := range stages {
		wg.Add(1)
		go func(stage Stage) {
			defer wg.Done()
			stage.Work()
		}(s)
	}
	wg.Wait()
	return time.Since(start)
}

// CalculateTimeSaving 计算并行执行节省的时间百分比
func CalculateTimeSaving(sequential, parallel time.Duration) float64 {
	if sequential == 0 {
		return 0
	}
	return float64(sequential-parallel) / float64(sequential) * 100
}
