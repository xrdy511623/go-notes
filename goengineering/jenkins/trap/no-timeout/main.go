package notimeout

/*
陷阱：Jenkins Pipeline 不设超时

问题说明：
  不设 timeout 的 Pipeline 一旦遇到阻塞，会无限挂起，直到 Jenkins 管理员手动终止。
  这不仅浪费 Agent 资源，还会阻塞后续构建（如果开启了 disableConcurrentBuilds）。

  常见的挂起场景：

  1. go test 死锁
     Go 测试中存在 goroutine 死锁，test 命令永不返回。
     go test 自身有 10 分钟默认超时（-timeout 10m），但不够可靠。

  2. 等待用户输入
     input 步骤没设超时，审批人忘了点击 → Pipeline 挂数天。

  3. 网络请求卡住
     curl / wget 下载外部依赖超时、Docker pull 卡住。

  4. Docker build 挂起
     Dockerfile 中某一层下载超时或死循环。

错误示例：

  pipeline {
      agent any
      stages {
          stage('Test') {
              steps {
                  sh 'go test -race ./...'  // 如果死锁，永远挂着
              }
          }
          stage('Approval') {
              steps {
                  input '部署到生产环境？'   // 没人审批就永远等着
              }
          }
      }
      // 没有 timeout！
  }

正确做法：

  pipeline {
      agent any
      options {
          timeout(time: 15, unit: 'MINUTES')   // 全局超时
      }
      stages {
          stage('Test') {
              options {
                  timeout(time: 10, unit: 'MINUTES')  // Stage 级超时
              }
              steps {
                  sh 'go test -race -timeout=8m ./...'  // Go 测试超时
              }
          }
          stage('Approval') {
              options {
                  timeout(time: 24, unit: 'HOURS')  // 审批超时
              }
              steps {
                  input '部署到生产环境？'
              }
          }
      }
  }

超时层次（从外到内）：
  1. Pipeline 全局 timeout（兜底）
  2. Stage 级 timeout（精细控制）
  3. go test -timeout（Go 测试层面）
  4. curl --max-time（网络请求层面）
*/

import (
	"fmt"
	"time"
)

// HangScenario 描述一种 Pipeline 挂起的场景
type HangScenario struct {
	Name        string
	Description string
	Fix         string
	Timeout     time.Duration
}

// CommonHangScenarios 列出常见的 Pipeline 挂起场景及解决方案
func CommonHangScenarios() []HangScenario {
	return []HangScenario{
		{
			Name:        "go test 死锁",
			Description: "goroutine 死锁导致 test 命令不返回",
			Fix:         "go test -race -timeout=8m ./...",
			Timeout:     8 * time.Minute,
		},
		{
			Name:        "Docker build 挂起",
			Description: "Dockerfile 层下载超时或死循环",
			Fix:         "timeout(time: 10, unit: 'MINUTES') 包裹 Docker stage",
			Timeout:     10 * time.Minute,
		},
		{
			Name:        "网络请求卡住",
			Description: "curl/wget 下载外部依赖无响应",
			Fix:         "curl --max-time 30 --retry 3",
			Timeout:     30 * time.Second,
		},
		{
			Name:        "input 审批无人响应",
			Description: "审批人忘记点击，Pipeline 挂数天",
			Fix:         "timeout(time: 24, unit: 'HOURS') 包裹 input 步骤",
			Timeout:     24 * time.Hour,
		},
		{
			Name:        "go mod download 卡住",
			Description: "私有模块代理不可达",
			Fix:         "设置 GONOSUMCHECK + GOPROXY + 超时",
			Timeout:     5 * time.Minute,
		},
	}
}

// RecommendedTimeouts 返回推荐的超时配置
func RecommendedTimeouts() map[string]time.Duration {
	return map[string]time.Duration{
		"Pipeline 全局":    15 * time.Minute,
		"Lint Stage":     5 * time.Minute,
		"Test Stage":     10 * time.Minute,
		"Build Stage":    5 * time.Minute,
		"Deploy Stage":   5 * time.Minute,
		"Approval Stage": 24 * time.Hour,
		"go test":        8 * time.Minute,
		"curl 请求":        30 * time.Second,
	}
}

// PrintHangScenarios 打印所有挂起场景及修复建议
func PrintHangScenarios() {
	fmt.Println("=== Jenkins Pipeline 挂起场景 ===")
	fmt.Println()
	for _, s := range CommonHangScenarios() {
		fmt.Printf("场景：%s\n", s.Name)
		fmt.Printf("  原因：%s\n", s.Description)
		fmt.Printf("  建议超时：%v\n", s.Timeout)
		fmt.Printf("  修复：%s\n\n", s.Fix)
	}

	fmt.Println("推荐超时配置：")
	for name, d := range RecommendedTimeouts() {
		fmt.Printf("  %-20s → %v\n", name, d)
	}
}
