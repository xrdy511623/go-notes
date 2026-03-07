package fatagent

/*
陷阱：所有工具装在一个大 Agent 上，不隔离

问题说明：
  很多团队在 Jenkins Agent 节点上把所有工具都装好：Go、Node.js、Java、
  Docker、kubectl、golangci-lint、protoc……变成一台"万能机器"。
  短期方便，长期灾难。

  问题：

  1. 版本冲突
     项目 A 需要 Go 1.23，项目 B 需要 Go 1.24。
     在同一 Agent 上只能装一个版本，要么手动切换，要么 symlink 地狱。

  2. 环境污染
     Job A 修改了全局 GOPATH 或安装了某个包，
     Job B 读到了意外的缓存 → 构建结果不可复现。

  3. 难以维护
     Agent 上装了几十个工具，升级一个可能破坏另一个。
     新增 Agent 需要手动装所有工具，耗时数小时。

  4. 安全隔离差
     所有 Job 共享同一文件系统、同一用户，
     一个 Job 写入的密钥文件另一个 Job 能读到。

错误示例：

  // 所有 Stage 都在同一个 agent any 上跑
  pipeline {
      agent any   // "万能" Agent，装了 Go + Node + Java + Docker + ...
      stages {
          stage('Build Go')   { steps { sh 'go build ...' } }
          stage('Build Node') { steps { sh 'npm run build' } }
          stage('Deploy')     { steps { sh 'kubectl apply ...' } }
      }
  }

正确做法：使用 Docker Agent 隔离

  pipeline {
      agent none  // 不指定全局 Agent
      stages {
          stage('Build Go') {
              agent {
                  docker {
                      image 'golang:1.24'
                      args '-v go-mod-cache:/go/pkg/mod'
                  }
              }
              steps { sh 'go build ...' }
          }
          stage('Deploy') {
              agent { label 'deployer' }  // 特定节点，有 kubectl 权限
              steps { sh 'kubectl apply ...' }
          }
      }
  }

  好处：
  - 每个 Stage 用精确版本的工具镜像
  - 构建环境完全隔离，不互相污染
  - 新增 Agent 只需安装 Docker，不需要装任何工具
  - 版本管理在 Jenkinsfile 里声明式控制
*/

import "fmt"

// AgentStrategy 描述一种 Agent 使用策略
type AgentStrategy struct {
	Name        string
	Description string
	Pros        []string
	Cons        []string
}

// CompareStrategies 对比 Fat Agent 和 Docker Agent 策略
func CompareStrategies() [2]AgentStrategy {
	return [2]AgentStrategy{
		{
			Name:        "Fat Agent（反模式）",
			Description: "所有工具安装在一个 Agent 节点上",
			Pros: []string{
				"初始设置简单",
				"不需要 Docker",
			},
			Cons: []string{
				"版本冲突（多项目需要不同 Go 版本）",
				"环境污染（Job 之间互相影响）",
				"维护成本高（每台 Agent 手动装工具）",
				"安全隔离差（共享文件系统）",
				"不可复现（Agent 状态随时间漂移）",
			},
		},
		{
			Name:        "Docker Agent（推荐）",
			Description: "每个 Stage 用独立 Docker 容器",
			Pros: []string{
				"完全隔离（每次干净环境）",
				"版本精确控制（镜像 tag）",
				"新增 Agent 只装 Docker",
				"可复现（镜像是不变的）",
				"安全隔离（容器文件系统隔离）",
			},
			Cons: []string{
				"需要 Docker 环境",
				"首次拉取镜像耗时（后续有缓存）",
			},
		},
	}
}

// PrintComparison 打印两种策略的对比
func PrintComparison() {
	fmt.Println("=== Jenkins Agent 策略对比 ===")
	fmt.Println()
	strategies := CompareStrategies()
	for _, s := range strategies {
		fmt.Printf("【%s】\n", s.Name)
		fmt.Printf("  %s\n\n", s.Description)
		fmt.Println("  优势：")
		for _, p := range s.Pros {
			fmt.Printf("    + %s\n", p)
		}
		fmt.Println("  劣势：")
		for _, c := range s.Cons {
			fmt.Printf("    - %s\n", c)
		}
		fmt.Println()
	}
	fmt.Println("结论：除非无法使用 Docker，否则始终用 Docker Agent")
}
