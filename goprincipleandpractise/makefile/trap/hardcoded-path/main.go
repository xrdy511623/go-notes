package hardcodedpath

/*
陷阱：Makefile 中硬编码路径

问题说明：
  在 Makefile 中硬编码绝对路径（如 GOPATH、用户目录、工具路径），
  会导致 Makefile 只能在特定机器上运行，换一台机器或换一个用户就会失败。

  常见的硬编码陷阱：

  1. 硬编码 GOPATH
     GOPATH := /Users/john/go         ← 只在 john 的机器上能用
     应该用: GOPATH := $(shell go env GOPATH)

  2. 硬编码工具路径
     LINT := /usr/local/bin/golangci-lint   ← 不同系统路径不同
     应该用: LINT := $(shell command -v golangci-lint)

  3. 硬编码构建输出路径
     OUTPUT := /opt/deploy/myapp       ← CI 环境没有 /opt/deploy
     应该用: OUTPUT := $(BUILD_DIR)/myapp

  4. 硬编码用户目录
     CONFIG := /home/deploy/.config    ← 不同用户路径不同
     应该用: CONFIG := $(HOME)/.config

后果：
  - 本地能 build，CI 上失败
  - 同事 clone 后 make 报错
  - 换 Mac/Linux 环境后失效
  - Docker 构建中路径不存在

正确做法：
  1. 用 $(shell ...) 动态获取路径
  2. 用相对路径而非绝对路径
  3. 用 ?= 允许环境变量覆盖
  4. 用 $(CURDIR) 代替硬编码的项目路径

  # 正确示例
  GOPATH      := $(shell go env GOPATH)
  GOBIN       := $(GOPATH)/bin
  BUILD_DIR   := $(CURDIR)/bin
  TOOLS_DIR   := $(CURDIR)/tools

  # 允许环境变量覆盖
  REGISTRY    ?= docker.io
  IMAGE_NAME  ?= myorg/myapp
*/

import "fmt"

// BadMakefileExample 展示硬编码路径的 Makefile 问题
func BadMakefileExample() string {
	return `# ❌ 错误示例：到处硬编码路径

GOPATH := /Users/john/go
GOBIN  := /Users/john/go/bin
LINT   := /usr/local/bin/golangci-lint
OUTPUT := /opt/deploy/myapp

build:
	$(GOBIN)/go build -o $(OUTPUT) .

lint:
	$(LINT) run ./...

deploy:
	scp $(OUTPUT) deploy@10.0.0.1:/opt/services/`
}

// GoodMakefileExample 展示使用动态路径的 Makefile
func GoodMakefileExample() string {
	return `# ✅ 正确示例：动态获取路径

GOPATH    := $(shell go env GOPATH)
GOBIN     := $(GOPATH)/bin
BUILD_DIR := $(CURDIR)/bin
LINT      := $(shell command -v golangci-lint 2>/dev/null)

# 允许环境变量覆盖
DEPLOY_HOST ?= deploy@staging.example.com
DEPLOY_DIR  ?= /opt/services

build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/myapp .

lint:
ifndef LINT
	@echo "golangci-lint not found, installing..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(eval LINT := $(GOBIN)/golangci-lint)
endif
	$(LINT) run ./...

deploy: build
	scp $(BUILD_DIR)/myapp $(DEPLOY_HOST):$(DEPLOY_DIR)/`
}

// CommonDynamicPaths 列出 Makefile 中常用的动态路径获取方式
func CommonDynamicPaths() map[string]string {
	return map[string]string{
		"GOPATH":      "$(shell go env GOPATH)",
		"GOBIN":       "$(shell go env GOPATH)/bin",
		"GOCACHE":     "$(shell go env GOCACHE)",
		"GOMODCACHE":  "$(shell go env GOMODCACHE)",
		"PROJECT_DIR": "$(CURDIR)",
		"GIT_ROOT":    "$(shell git rev-parse --show-toplevel)",
		"USER_HOME":   "$(HOME)",
		"OS":          "$(shell uname -s)",
		"ARCH":        "$(shell uname -m)",
		"GO_VERSION":  "$(shell go version | awk '{print $$3}')",
		"TOOL_PATH":   "$(shell command -v <tool> 2>/dev/null)",
	}
}

// PrintDynamicPaths 打印所有动态路径建议
func PrintDynamicPaths() {
	fmt.Println("=== Makefile 动态路径最佳实践 ===")
	fmt.Println()
	for name, expr := range CommonDynamicPaths() {
		fmt.Printf("  %-15s := %s\n", name, expr)
	}
	fmt.Println()
	fmt.Println("规则：永远不要在 Makefile 中写死绝对路径")
}
