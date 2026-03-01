package nocleanup

/*
陷阱：Jenkins Pipeline 不清理工作空间

问题说明：
  Jenkins 默认会保留工作空间（workspace）内容，不会在构建完成后自动清理。
  如果 Pipeline 没有在 post 块中调用 cleanWs()，残留文件会导致以下问题：

  1. 磁盘空间耗尽
     每次构建残留 Go 编译产物（bin/）、测试覆盖率报告（coverage.out）、
     临时文件等。节点磁盘逐渐被填满，最终导致构建失败。

  2. 脏状态污染下次构建
     上次构建留下的 coverage.out 可能让覆盖率门禁误判。
     上次编译的二进制还在 bin/ 里，看起来"构建成功"但实际没有重新编译。
     上次的 go.sum 冲突导致 go mod tidy 失败。

  3. 凭据残留（安全风险）
     withCredentials 写入的临时文件（如 kubeconfig）可能留在磁盘上。
     后续无权限的 Job 如果共享同一节点，可能读取到这些文件。

错误示例：

  pipeline {
      agent any
      stages {
          stage('Build') {
              steps {
                  sh 'go build -o bin/app ./cmd/...'
              }
          }
      }
      // 没有 post 块！工作空间永不清理
  }

正确做法：

  pipeline {
      agent any
      stages {
          stage('Build') {
              steps {
                  sh 'go build -o bin/app ./cmd/...'
              }
          }
      }
      post {
          always {
              cleanWs()  // 无论成功失败都清理
          }
      }
  }

  如果只想清理特定文件：

  post {
      always {
          sh 'rm -rf bin/ coverage.out *.tmp'
          // 或使用 cleanWs 的模式匹配
          cleanWs(patterns: [
              [pattern: 'bin/**', type: 'INCLUDE'],
              [pattern: 'coverage.out', type: 'INCLUDE']
          ])
      }
  }
*/

import "fmt"

// WorkspaceRisks 列出不清理工作空间的风险
func WorkspaceRisks() []Risk {
	return []Risk{
		{
			Category:    "磁盘空间",
			Description: "编译产物、测试报告、日志逐次累积",
			Impact:      "节点磁盘满 → 所有 Job 失败",
			Severity:    "HIGH",
		},
		{
			Category:    "构建污染",
			Description: "上次残留文件影响本次构建结果",
			Impact:      "假阳性/假阴性 → 信任危机",
			Severity:    "HIGH",
		},
		{
			Category:    "安全风险",
			Description: "凭据临时文件残留在磁盘",
			Impact:      "密钥泄漏 → 越权访问",
			Severity:    "CRITICAL",
		},
		{
			Category:    "Go 特有",
			Description: "go.sum 冲突、vendor/ 残留",
			Impact:      "go mod tidy 失败 → 构建中断",
			Severity:    "MEDIUM",
		},
	}
}

// Risk 描述一种不清理工作空间的风险
type Risk struct {
	Category    string
	Description string
	Impact      string
	Severity    string
}

// PrintRisks 打印所有风险
func PrintRisks() {
	fmt.Println("=== Jenkins 不清理工作空间的风险 ===")
	fmt.Println()
	for _, r := range WorkspaceRisks() {
		fmt.Printf("[%s] %s\n", r.Severity, r.Category)
		fmt.Printf("  问题：%s\n", r.Description)
		fmt.Printf("  后果：%s\n\n", r.Impact)
	}
	fmt.Println("解决方案：在 post { always { cleanWs() } } 中清理")
}

// CleanupStrategies 列出工作空间清理策略
func CleanupStrategies() map[string]string {
	return map[string]string{
		"cleanWs()":              "清理整个工作空间（推荐）",
		"deleteDir()":            "删除当前目录（旧 API）",
		"sh 'rm -rf bin/ *.out'": "手动清理特定文件",
		"cleanWs(patterns: ...)": "按模式匹配清理",
	}
}
