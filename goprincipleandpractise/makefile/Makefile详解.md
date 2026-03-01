# Go 项目 Makefile 设计详解

> Makefile 是 Go 项目的统一构建入口，让 `go build`、`go test`、`golangci-lint` 等命令标准化，新人一个 `make help` 就能上手。

## 目录

1. [为什么 Go 项目需要 Makefile](#1-为什么-go-项目需要-makefile)
2. [核心 target 设计](#2-核心-target-设计)
3. [变量与约定](#3-变量与约定)
4. [多平台交叉编译](#4-多平台交叉编译)
5. [.PHONY 与依赖链](#5-phony-与依赖链)
6. [与 go generate 集成](#6-与-go-generate-集成)
7. [进阶技巧](#7-进阶技巧)
8. [依赖工具与环境自检](#8-依赖工具与环境自检)

---

## 1 为什么 Go 项目需要 Makefile

Go 自带 `go build`、`go test` 等工具链，为什么还需要 Makefile？

| 场景 | 直接用 go 命令 | 用 Makefile |
|------|---------------|-------------|
| 新人入职 | 需读文档找构建命令 | `make help` 一目了然 |
| CI 流水线 | 每个 step 写完整命令 | `make lint test build` |
| 多平台构建 | 手动设 GOOS/GOARCH | `make build-all` |
| 版本注入 | 记住 ldflags 格式 | `make build` 自动注入 |
| 一致性 | 不同开发者命令不同 | 统一入口，结果可复现 |

**核心价值**：Makefile 是"构建知识"的沉淀，把零散命令变成可复用、可审查的流程。

---

## 2 核心 target 设计

### 推荐的标准 target

```makefile
# 项目元信息
APP_NAME    := myapp
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME  := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
BUILD_DIR   := bin

# Go 相关
GOFLAGS     := -trimpath
LDFLAGS     := -s -w \
  -X 'main.version=$(VERSION)' \
  -X 'main.commit=$(COMMIT)' \
  -X 'main.buildTime=$(BUILD_TIME)'

.PHONY: all build test lint fmt vet clean run cover help

## all: 默认目标 — lint + test + build
all: lint test build

## build: 编译二进制到 bin/
build:
	@echo "Building $(APP_NAME) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) ./cmd/$(APP_NAME)

## test: 运行测试（含 race 检测）
test:
	go test -race -count=1 ./...

## lint: 运行 golangci-lint
lint:
	golangci-lint run --timeout=5m ./...

## fmt: 格式化代码
fmt:
	gofmt -w .
	goimports-reviser -w .

## vet: 静态分析
vet:
	go vet ./...

## cover: 测试覆盖率
cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@echo "---"
	@echo "HTML report: go tool cover -html=coverage.out"

## clean: 清理构建产物
clean:
	rm -rf $(BUILD_DIR) coverage.out

## run: 编译并运行
run: build
	./$(BUILD_DIR)/$(APP_NAME)

## help: 显示帮助信息
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'
```

### target 职责说明

| Target | 职责 | CI 中使用 | 本地使用 |
|--------|------|-----------|---------|
| `build` | 编译二进制 | ✅ | ✅ |
| `test` | 运行测试 + race 检测 | ✅ | ✅ |
| `lint` | 代码质量检查 | ✅ | ✅ |
| `fmt` | 代码格式化 | ❌ | ✅ |
| `vet` | 静态分析 | ✅ | ✅ |
| `cover` | 覆盖率报告 | ✅ | ✅ |
| `clean` | 清理产物 | ❌ | ✅ |
| `run` | 编译并运行 | ❌ | ✅ |
| `help` | 显示帮助 | ❌ | ✅ |

### 本地开发 vs CI 分层 target（推荐）

下面这组 target 可以解决“本地反馈慢、CI 规则重”的常见问题：

```makefile
.PHONY: pre-commit test-fast ci

## test-fast: 本地快速测试（不带 race）
test-fast:
	go test -count=1 ./...

## pre-commit: 提交前检查（本地）
pre-commit: fmt vet test-fast

## ci: CI 全量门禁
ci: lint test build
```

建议：
- 本地日常用 `make test-fast` 和 `make pre-commit`，缩短反馈时间
- CI 流水线固定用 `make ci`，确保质量标准一致

---

## 3 变量与约定

### 核心变量

```makefile
# === 项目信息 ===
APP_NAME    := myapp                    # 二进制名称
BUILD_DIR   := bin                      # 输出目录
CMD_DIR     := ./cmd/$(APP_NAME)        # 入口目录

# === 版本信息（自动从 Git 获取）===
# 推荐加 fallback，避免在无 git 环境（如源码包、部分 CI 容器）中失败
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME  := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

# === Go 编译选项 ===
GOFLAGS     := -trimpath               # 去除本地路径信息
CGO_ENABLED ?= 0                       # 默认禁用 CGO（纯静态编译）

# === ldflags（链接器标志）===
LDFLAGS     := -s -w                    # -s 去符号表, -w 去 DWARF
LDFLAGS     += -X 'main.version=$(VERSION)'
LDFLAGS     += -X 'main.commit=$(COMMIT)'
LDFLAGS     += -X 'main.buildTime=$(BUILD_TIME)'

# === 工具版本 ===
LINT_VERSION := v1.62.2
```

### 变量覆盖

```bash
# 使用者可以覆盖变量
make build CGO_ENABLED=1        # 启用 CGO
make build VERSION=v2.0.0       # 手动指定版本
make build BUILD_DIR=output     # 修改输出目录
```

`?=` 赋值允许环境变量覆盖：

```makefile
CGO_ENABLED ?= 0  # 如果环境变量已设置，使用环境变量的值
```

---

## 4 多平台交叉编译

```makefile
# 支持的平台列表
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

## build-all: 交叉编译所有平台
build-all:
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} \
		EXT=""; \
		if [ "$${platform%/*}" = "windows" ]; then EXT=".exe"; fi; \
		echo "Building $${platform}..."; \
		CGO_ENABLED=0 GOOS=$${platform%/*} GOARCH=$${platform#*/} \
			go build $(GOFLAGS) -ldflags="$(LDFLAGS)" \
			-o $(BUILD_DIR)/$(APP_NAME)-$${platform%/*}-$${platform#*/}$${EXT} \
			$(CMD_DIR); \
	done

## build-linux: 仅构建 Linux 版本
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build $(GOFLAGS) -ldflags="$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 $(CMD_DIR)

## build-darwin: 仅构建 macOS 版本（Apple Silicon）
build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 \
		go build $(GOFLAGS) -ldflags="$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 $(CMD_DIR)
```

**交叉编译的关键**：`CGO_ENABLED=0` 是纯 Go 交叉编译的前提。启用 CGO 时需要对应平台的 C 编译器（如 `aarch64-linux-gnu-gcc`），复杂度剧增。

---

## 5 .PHONY 与依赖链

### 为什么需要 .PHONY

Make 的原始设计是基于文件的：target 是文件名，如果文件已存在且比依赖更新，就跳过执行。

**问题**：如果目录下碰巧有个文件叫 `build` 或 `test`，make 会认为目标已达成，跳过执行。

```makefile
# 必须声明 .PHONY，否则同名文件会导致 target 不执行
.PHONY: build test lint fmt vet clean run cover help all
```

> **反例**: [trap/no-phony/](trap/no-phony/) — 不声明 .PHONY 导致 target 被跳过

### 依赖链设计

```makefile
# 依赖关系：
# all → lint, test, build
# release → all, build-all
# deploy → release

.PHONY: all release deploy

all: lint test build

release: all build-all
	@echo "Release $(VERSION) ready in $(BUILD_DIR)/"

deploy: release
	@echo "Deploying $(VERSION)..."
```

**执行顺序**：`make deploy` 会按依赖链自动执行 `lint → test → build → build-all → deploy`。

---

## 6 与 go generate 集成

```makefile
## generate: 运行代码生成
generate:
	go generate ./...

## proto: 编译 protobuf
proto:
	protoc --go_out=. --go-grpc_out=. proto/*.proto

## mock: 生成 mock 文件
mock:
	mockgen -source=internal/repository/user.go \
		-destination=internal/repository/mock/user_mock.go

## build: 先生成代码再编译
build: generate
	CGO_ENABLED=0 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(APP_NAME) $(CMD_DIR)
```

---

## 7 进阶技巧

### 7.1 help target 自动生成

利用注释自动生成帮助信息：

```makefile
## help: 显示所有可用 target
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ": "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' | \
		sed 's/## //'
```

输出效果：
```
Usage: make [target]

Targets:
  all                  默认目标 — lint + test + build
  build                编译二进制到 bin/
  test                 运行测试（含 race 检测）
  lint                 运行 golangci-lint
  ...
```

### 7.2 条件判断

```makefile
# 检查工具是否安装
LINT := $(shell command -v golangci-lint 2>/dev/null)

lint:
ifndef LINT
	@echo "Installing golangci-lint..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(LINT_VERSION)
endif
	golangci-lint run --timeout=5m ./...
```

### 7.3 颜色输出

```makefile
GREEN  := \033[32m
YELLOW := \033[33m
RED    := \033[31m
RESET  := \033[0m

build:
	@echo "$(GREEN)Building $(APP_NAME) $(VERSION)...$(RESET)"
	@CGO_ENABLED=0 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(APP_NAME) $(CMD_DIR)
	@echo "$(GREEN)✓ Build complete: $(BUILD_DIR)/$(APP_NAME)$(RESET)"

test:
	@echo "$(YELLOW)Running tests...$(RESET)"
	@go test -race -count=1 ./... && \
		echo "$(GREEN)✓ All tests passed$(RESET)" || \
		(echo "$(RED)✗ Tests failed$(RESET)" && exit 1)
```

---

## 8 依赖工具与环境自检

文档中的一些 target 依赖外部工具，建议在项目中提供 `deps`/`check-tools`，减少“照抄失败”。

```makefile
.PHONY: deps check-tools

## deps: 安装常用开发工具（按需调整版本）
deps:
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.2
	go install github.com/golang/mock/mockgen@latest

## check-tools: 检查工具是否可用
check-tools:
	@command -v go >/dev/null || (echo "missing: go" && exit 1)
	@command -v golangci-lint >/dev/null || (echo "missing: golangci-lint" && exit 1)
	@command -v goimports >/dev/null || (echo "missing: goimports" && exit 1)
	@command -v mockgen >/dev/null || (echo "missing: mockgen" && exit 1)
	@command -v protoc >/dev/null || (echo "missing: protoc (optional, needed by make proto)" && exit 1)
```

实践建议：
- 在 `help` 中显式展示 `deps`、`check-tools`
- 在 CI 首步执行 `make check-tools`
- `proto`/`mock` 这类可选 target，在说明里写清“何时需要”

---

## 总结

| 要点 | 说明 |
|------|------|
| 标准 target | build, test, lint, fmt, vet, cover, clean, run, help |
| .PHONY | 所有非文件 target 都必须声明 |
| 版本注入 | 通过 ldflags 自动从 Git 获取 |
| 交叉编译 | CGO_ENABLED=0 + GOOS/GOARCH 循环 |
| help | 利用注释自动生成，新人友好 |
| CI 复用 | `make lint test build` 一行搞定 |

**常见陷阱**：
- 不声明 .PHONY：[trap/no-phony/](trap/no-phony/)
- 硬编码路径：[trap/hardcoded-path/](trap/hardcoded-path/)
- 不检查错误：[trap/no-error-check/](trap/no-error-check/)

**性能对比**：
- 并行构建：[performance/parallel-build/](performance/parallel-build/)
- 增量构建 vs 全量构建：[performance/incremental-vs-full/](performance/incremental-vs-full/)
