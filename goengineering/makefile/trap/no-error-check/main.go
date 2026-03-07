package noerrorcheck

/*
陷阱：Makefile target 中不检查命令退出码

问题说明：
  Makefile 中的每一行 recipe 默认在独立的 shell 中执行。
  如果某一行命令失败（退出码非 0），Make 会停止执行后续行。

  但有几种常见情况会导致错误被静默忽略：

  1. 使用 - 前缀忽略错误
     -rm -rf build/     # - 前缀让 Make 忽略失败
     go build .         # 即使上面失败也会继续

  2. 用 || true 吞掉错误
     go test ./... || true    # 测试失败也"成功"
     echo "Tests passed!"     # 永远会执行到这里

  3. 用 ; 而不是 && 连接命令
     cd subdir; go build .    # 如果 cd 失败，go build 在错误目录执行
     应该用: cd subdir && go build .

  4. 管道命令中间环节失败
     go test ./... | tee output.log    # 如果 go test 失败，退出码是 tee 的
     应该加: SHELL := /bin/bash 和 .SHELLFLAGS := -o pipefail -c

后果：
  - CI 流水线"绿灯"但实际测试失败
  - 构建产物不完整但部署继续
  - 错误被吞掉，问题累积到生产环境

正确做法：
  1. 不要用 || true 吞掉关键命令的错误
  2. 用 && 连接有依赖关系的命令
  3. 开启 pipefail：
     SHELL := /bin/bash
     .SHELLFLAGS := -o pipefail -c
  4. 只在确实需要忽略的地方用 - 前缀（如 clean 中的 rm）
*/

import "fmt"

// BadTarget 展示错误吞掉的 Makefile target
func BadTarget() string {
	return `# ❌ 错误示例：测试失败被吞掉

test:
	go test ./... || true
	@echo "All tests passed!"

# 问题：
# 即使测试失败，也会输出 "All tests passed!"
# CI 看到退出码 0，标记为绿灯
# 实际上有测试失败了`
}

// BadPipeline 展示管道命令中错误丢失的问题
func BadPipeline() string {
	return `# ❌ 错误示例：管道中间失败被忽略

test-report:
	go test -json ./... | go-test-report > report.html

# 问题：
# 如果 go test 失败（退出码 1），管道的退出码是最后一个命令的
# go-test-report 成功退出（退出码 0），整个命令"成功"
# Make 认为 target 成功，CI 绿灯`
}

// GoodTarget 展示正确的错误处理
func GoodTarget() string {
	return `# ✅ 正确示例：错误正确传播

SHELL := /bin/bash
.SHELLFLAGS := -o pipefail -c

test:
	go test -race -count=1 ./...

test-report:
	go test -json ./... | go-test-report > report.html

# 开启 pipefail 后：
# 管道中任何一个命令失败，整个管道就失败
# go test 失败 → Make 停止 → CI 红灯`
}

// SilentFailureScenarios 列出容易导致静默失败的场景
func SilentFailureScenarios() []string {
	return []string{
		"go test ./... || true — 测试失败被吞掉",
		"cd dir; command — cd 失败后在错误目录执行",
		"command | tee log — 管道前段失败被忽略",
		"-command — 任何失败都被忽略",
		"command 2>/dev/null — 错误信息被丢弃",
		"command; echo done — 分号连接不检查前序命令",
	}
}

// PrintScenarios 打印所有静默失败场景
func PrintScenarios() {
	fmt.Println("=== Makefile 静默失败场景 ===")
	fmt.Println()
	for i, s := range SilentFailureScenarios() {
		fmt.Printf("  %d. %s\n", i+1, s)
	}
	fmt.Println()
	fmt.Println("规则：关键命令的退出码必须正确传播到 Make")
}
