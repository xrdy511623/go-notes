# CI/CD 与 PR 流程详解

> 覆盖 GitHub Actions、GitLab CI 配置、PR 最佳实践、Pipeline 设计模式，配有反例（trap/）和性能基准（performance/）。

> 说明：本仓库当前未内置 `.github/workflows/` 或 `.gitlab-ci.yml`，下文配置为可迁移模板（示例），用于落地时参考。

## 目录

1. [CI/CD 基本概念](#1-cicd-基本概念)
2. [GitHub Actions 详解](#2-github-actions-详解)
3. [GitLab CI 对比](#3-gitlab-ci-对比)
4. [PR 流程最佳实践](#4-pr-流程最佳实践)
5. [Pipeline 设计模式](#5-pipeline-设计模式)
6. [Go 项目 CI 最佳实践](#6-go-项目-ci-最佳实践)

---

## 1 CI/CD 基本概念

| 概念 | 含义 | 触发时机 | 自动化程度 |
|------|------|---------|-----------|
| 持续集成（CI） | 代码合并后自动构建+测试 | 每次 push / PR | 全自动 |
| 持续交付（CD） | CI + 自动部署到 staging | CI 通过后 | 部署需手动审批 |
| 持续部署（CD） | CI + 自动部署到生产 | CI 通过后 | 全自动（含生产） |

```
开发者 push ──→ CI（lint+test+build）──→ CD（staging）──→ CD（production）
                    │                        │                  │
                    └─ 自动                   └─ 自动/手动       └─ 手动审批
```

**核心目标**：快速反馈。流水线应在 5-10 分钟内给出结果。

---

## 2 GitHub Actions 详解

### 2.1 Workflow 文件结构

```yaml
# .github/workflows/ci.yml
name: CI                           # workflow 名称

on:                                # 触发条件
  push:
    branches: [main]
  pull_request:
    branches: [main]

permissions:                       # 最小权限原则
  contents: read

env:                               # 全局环境变量
  GO_VERSION: '1.24'
  GOLANGCI_LINT_VERSION: 'v1.62.2' # 建议固定版本，避免 latest 漂移

jobs:                              # 任务定义
  test:
    name: Test
    runs-on: ubuntu-latest         # 运行环境
    steps:
      - uses: actions/checkout@v4  # 检出代码
      - uses: actions/setup-go@v5  # 安装 Go
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true              # 缓存 go mod + build cache
      - run: go test -race ./...   # 执行命令
```

### 2.2 常用触发器

```yaml
on:
  push:
    branches: [main, develop]
    tags: ['v*']
    paths:
      - '**.go'
      - 'go.mod'
      - 'go.sum'
    paths-ignore:
      - '**.md'
      - 'docs/**'

  pull_request:
    branches: [main]
    types: [opened, synchronize, reopened]

  schedule:
    - cron: '0 2 * * 1'           # 每周一凌晨 2 点

  workflow_dispatch:               # 手动触发
    inputs:
      environment:
        description: 'Deploy environment'
        required: true
        default: 'staging'
        type: choice
        options: [staging, production]
```

### 2.3 Go 项目标准 Workflow

```yaml
name: Go CI

on:
  push:
    branches: [main]
    paths: ['**.go', 'go.mod', 'go.sum']
  pull_request:
    branches: [main]

permissions:
  contents: read

env:
  GO_VERSION: '1.24'

jobs:
  lint:
    name: Lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
      - uses: golangci/golangci-lint-action@v6
        with:
          version: ${{ env.GOLANGCI_LINT_VERSION }}
          args: --timeout=5m

  test:
    name: Test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true
      - run: go test -race -coverprofile=coverage.out -covermode=atomic ./...
      - name: Coverage gate
        run: |
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          echo "Coverage: ${COVERAGE}%"
          if ! awk -v cov="$COVERAGE" 'BEGIN { exit (cov+0 >= 80) ? 0 : 1 }'; then
            echo "::error::覆盖率 ${COVERAGE}% 低于 80% 门禁"
            exit 1
          fi

  build:
    name: Build
    needs: [lint, test]
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true
      - run: CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/app ./cmd/...
      - uses: actions/upload-artifact@v4
        with:
          name: binary
          path: bin/
          retention-days: 7
```

### 2.4 Matrix 策略

```yaml
jobs:
  test:
    strategy:
      matrix:
        go-version: ['1.23', '1.24']
        os: [ubuntu-latest, macos-latest]
      fail-fast: false               # 一个失败不影响其他
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ matrix.go-version }}
      - run: go test -race ./...
```

### 2.5 缓存策略

```yaml
# 方式一：setup-go 内置缓存（推荐）
- uses: actions/setup-go@v5
  with:
    go-version: '1.24'
    cache: true                    # 自动缓存 GOMODCACHE + GOCACHE

# 方式二：手动缓存（更精细控制）
- uses: actions/cache@v4
  with:
    path: |
      ~/go/pkg/mod
      ~/.cache/go-build
    key: go-${{ runner.os }}-${{ hashFiles('**/go.sum') }}
    restore-keys: |
      go-${{ runner.os }}-
```

> **反例**: [trap/no-cache/](trap/no-cache/) — 不配缓存导致每次 CI 多花 2-5 分钟
>
> **性能对比**: [performance/cache-vs-nocache/](performance/cache-vs-nocache/) — 缓存 vs 无缓存的构建时间对比

---

## 3 GitLab CI 对比

### 3.1 .gitlab-ci.yml 基本结构

```yaml
# .gitlab-ci.yml
image: golang:1.24

variables:
  GOPATH: /go
  CGO_ENABLED: "0"
  GOLANGCI_LINT_VERSION: "v1.62.2"

cache:
  key: ${CI_COMMIT_REF_SLUG}
  paths:
    - /go/pkg/mod/

stages:
  - lint
  - test
  - build
  - deploy

lint:
  stage: lint
  script:
    - go install github.com/golangci/golangci-lint/cmd/golangci-lint@${GOLANGCI_LINT_VERSION}
    - golangci-lint run --timeout=5m ./...

test:
  stage: test
  script:
    - go test -race -coverprofile=coverage.out ./...
    - go tool cover -func=coverage.out
  coverage: '/total:\s+\(statements\)\s+(\d+\.\d+)%/'

build:
  stage: build
  script:
    - go build -ldflags="-s -w" -o bin/app ./cmd/...
  artifacts:
    paths:
      - bin/app
    expire_in: 1 week
  rules:
    - if: '$CI_COMMIT_BRANCH == "main"'
    - if: '$CI_COMMIT_TAG'
```

### 3.2 GitHub Actions vs GitLab CI 核心差异

| 维度 | GitHub Actions | GitLab CI |
|------|---------------|-----------|
| 配置文件 | `.github/workflows/*.yml`（多文件） | `.gitlab-ci.yml`（单文件） |
| 执行单元 | Job → Step | Stage → Job → Script |
| 并行控制 | job 默认并行，step 串行 | 同 stage 内 job 并行 |
| 缓存 | `actions/cache` 或内置 | 内置 `cache:` 关键字 |
| 制品 | `actions/upload-artifact` | `artifacts:` 关键字 |
| 触发条件 | `on:` + 复杂事件匹配 | `rules:`（推荐）/ `only:` / `except:` |
| 复用 | Reusable workflows | `include:` / `extends:` |
| 生态 | Marketplace（第三方丰富） | 模板库 |
| 容器支持 | 需 Docker action | 原生 Docker-in-Docker |

---

## 4 PR 流程最佳实践

### 4.1 Branch 策略

**Trunk-Based Development（推荐）**：
```
main ─────●────●────●────●────●──→
          │         │
          └─ feat ──┘  (短生命周期，< 2 天)
```

**Git-Flow**：
```
main     ─────────────●──────────●──→
develop  ●──●──●──●──●──●──●──●──→
          ↑     ↑
feature   └──●──┘
```

| 策略 | 适用场景 | 复杂度 |
|------|---------|--------|
| Trunk-Based | 持续部署，CI/CD 成熟 | 低 |
| Git-Flow | 版本化发布 | 高 |
| 不确定 | 先用 Trunk-Based | — |

### 4.2 PR 模板

```markdown
## 变更说明
<!-- 描述变更内容和原因 -->

## 变更类型
- [ ] 新功能
- [ ] Bug 修复
- [ ] 重构
- [ ] 文档更新

## 测试
- [ ] 单元测试已通过
- [ ] 已运行 `go test -race ./...`

## Checklist
- [ ] 代码符合规范
- [ ] 无硬编码敏感信息
- [ ] commit message 符合 conventional commits
```

### 4.3 Code Review Checklist

**必查项（CRITICAL）**：
- 安全漏洞（SQL 注入、硬编码 secret）
- 错误处理（不吞错误、不 panic）
- 并发安全（race condition、死锁）
- 资源泄漏（goroutine、连接）

**应查项（HIGH）**：
- 代码可读性
- 测试覆盖率 ≥ 80%
- 性能考量（N+1 查询）
- 接口设计

### 4.4 合并策略

| 策略 | 历史 | 适用场景 |
|------|------|---------|
| Merge Commit | 保留所有 commit | 需要完整历史 |
| **Squash Merge** | 一个 PR 一个 commit | **推荐默认** |
| Rebase Merge | 线性历史 | 追求干净历史 |

### 4.5 保护分支配置

```
main 分支保护规则：
  Require pull request reviews（至少 1 人）
  Require status checks to pass（lint + test）
  Require branches to be up to date
  Require linear history（强制 squash/rebase）
  禁止 force push
  禁止删除
```

---

## 5 Pipeline 设计模式

### 5.1 标准流水线

```
┌──────┐   ┌──────┐   ┌───────┐   ┌────────┐   ┌────────┐
│ Lint │──▶│ Test │──▶│ Build │──▶│ Deploy │──▶│ Verify │
│      │   │      │   │       │   │Staging │   │  E2E   │
└──────┘   └──────┘   └───────┘   └────────┘   └────────┘
                                       │
                                       ▼ (手动审批)
                                  ┌────────┐
                                  │ Deploy │
                                  │  Prod  │
                                  └────────┘
```

### 5.2 并行 Job 设计

无依赖关系的 job 应并行执行：

```yaml
jobs:
  lint:        # ─┐
    ...        #  ├─ 并行执行
  unit-test:   #  │
    ...        #  │
  security:    # ─┘
    ...

  build:                          # 全部通过后
    needs: [lint, unit-test, security]
    ...
```

> **性能对比**: [performance/parallel-jobs/](performance/parallel-jobs/) — 串行 vs 并行流水线时间对比

### 5.3 条件执行

```yaml
on:
  push:
    paths:
      - '**.go'
      - 'go.mod'
      - 'go.sum'
    paths-ignore:
      - '**.md'
      - 'docs/**'

jobs:
  deploy:
    if: github.ref == 'refs/heads/main' && github.event_name == 'push'

  release:
    if: startsWith(github.ref, 'refs/tags/v')
```

---

## 6 Go 项目 CI 最佳实践

### 6.1 静态分析三件套

```yaml
- run: go vet ./...                        # 内置检查
- run: staticcheck ./...                    # 扩展静态分析
- uses: golangci/golangci-lint-action@v6    # 聚合 linter
```

**推荐 `.golangci.yml` 配置**：

```yaml
linters:
  enable:
    - errcheck       # 未处理的错误
    - gosimple       # 简化建议
    - govet          # go vet
    - staticcheck    # 高级静态分析
    - unused         # 未使用代码
    - bodyclose      # HTTP body 未关闭
    - gosec          # 安全问题
    - prealloc       # slice 预分配
    - misspell       # 拼写错误

run:
  timeout: 5m
```

> **反例**: [trap/no-lint-in-ci/](trap/no-lint-in-ci/) — 不配 linter 遗漏 shadow error、未检查 error

### 6.2 测试覆盖率门禁

```yaml
- run: go test -race -coverprofile=coverage.out -covermode=atomic ./...
- run: |
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
    if ! awk -v cov="$COVERAGE" 'BEGIN { exit (cov+0 >= 80) ? 0 : 1 }'; then
      echo "::error::覆盖率 ${COVERAGE}% 低于 80% 门禁"
      exit 1
    fi
```

### 6.3 Race Detection

**CI 中必须开启 `-race`**：

```yaml
- run: go test -race ./...
  env:
    GORACE: "halt_on_error=1"
```

注意：`-race` 使测试变慢 2-10x，内存增加 5-10x，但绝对值得。

> **反例**: [trap/test-without-race/](trap/test-without-race/) — 不加 -race 导致隐藏的数据竞争

### 6.4 Benchmark 回归检测

```yaml
- run: go test -run='^$' -bench=. -benchmem -count=5 ./... > new.txt
- run: |
    # 固定版本或 commit，避免 latest 漂移
    go install golang.org/x/perf/cmd/benchstat@<pinned-version>
    benchstat baseline.txt new.txt
```

### 6.5 完整 CI 配置示例

```yaml
name: Go CI

on:
  push:
    branches: [main]
    paths: ['**.go', 'go.mod', 'go.sum', '.github/workflows/ci.yml']
  pull_request:
    branches: [main]

permissions:
  contents: read

env:
  GO_VERSION: '1.24'
  GOLANGCI_LINT_VERSION: 'v1.62.2'
  GOSEC_VERSION: 'v2.22.0'

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '${{ env.GO_VERSION }}' }
      - uses: golangci/golangci-lint-action@v6
        with: { version: '${{ env.GOLANGCI_LINT_VERSION }}', args: '--timeout=5m' }

  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go-version: ['1.23', '1.24']
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '${{ matrix.go-version }}', cache: true }
      - run: go test -race -coverprofile=coverage.out -covermode=atomic ./...
        env: { GORACE: 'halt_on_error=1' }
      - name: Coverage gate
        if: matrix.go-version == '1.24'
        run: |
          COV=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          echo "Coverage: ${COV}%"
          awk -v cov="$COV" 'BEGIN { exit (cov+0 >= 80) ? 0 : 1 }' || (echo "::error::Low coverage" && exit 1)

  security:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '${{ env.GO_VERSION }}' }
      - run: go install github.com/securego/gosec/v2/cmd/gosec@${{ env.GOSEC_VERSION }} && gosec ./...

  build:
    needs: [lint, test, security]
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '${{ env.GO_VERSION }}', cache: true }
      - run: CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/app ./cmd/...
      - uses: actions/upload-artifact@v4
        with: { name: binary, path: bin/, retention-days: 7 }
```

---

## 总结

| 实践 | 关键点 |
|------|--------|
| CI 流水线 | lint → test(-race) → build，缺一不可 |
| 缓存 | 必须配置 go mod + build cache |
| 覆盖率 | 80% 门禁，自动化检查 |
| Race 检测 | CI 中必须开启 -race |
| 并行 | 无依赖 job 并行执行，缩短反馈时间 |
| PR 流程 | 模板 + checklist + 保护分支 + squash merge |

**常见陷阱**：
- 不配缓存：[trap/no-cache/](trap/no-cache/)
- 不加 race 检测：[trap/test-without-race/](trap/test-without-race/)
- 不配 linter：[trap/no-lint-in-ci/](trap/no-lint-in-ci/)

**性能对比**：
- 缓存 vs 无缓存：[performance/cache-vs-nocache/](performance/cache-vs-nocache/)
- 串行 vs 并行流水线：[performance/parallel-jobs/](performance/parallel-jobs/)
