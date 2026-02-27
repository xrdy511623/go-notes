# Go 版本管理与发布流程详解

> 覆盖语义化版本、ldflags 注入、goreleaser 自动发布、viper 配置管理、Go module 发布，配有反例（trap/）和性能基准（performance/）。

## 目录

1. [语义化版本](#1-语义化版本)
2. [构建时注入版本信息（ldflags）](#2-构建时注入版本信息ldflags)
3. [goreleaser 详解](#3-goreleaser-详解)
4. [配置管理](#4-配置管理)
5. [发布流程自动化](#5-发布流程自动化)
6. [Go module 发布](#6-go-module-发布)

---

## 1 语义化版本

### 1.1 MAJOR.MINOR.PATCH

| 位置 | 含义 | 何时递增 | 示例 |
|------|------|---------|------|
| MAJOR | 主版本 | 不兼容的 API 变更 | 1.0.0 → 2.0.0 |
| MINOR | 次版本 | 向后兼容的功能新增 | 1.0.0 → 1.1.0 |
| PATCH | 补丁版本 | 向后兼容的 bug 修复 | 1.0.0 → 1.0.1 |

### 1.2 Pre-release 与 Build metadata

```
v1.2.3-alpha.1       # 预发布版本
v1.2.3-beta.2        # beta 阶段
v1.2.3-rc.1          # 发布候选
v1.2.3+build.123     # 构建元数据（不参与排序）
```

排序：`1.0.0-alpha < 1.0.0-beta < 1.0.0-rc.1 < 1.0.0`

### 1.3 Go module 版本与 SemVer

**v0.x.x — 不稳定**：API 可随时变更
```go
require github.com/example/lib v0.3.2
```

**v1.x.x — 稳定**：遵循兼容性保证
```go
require github.com/example/lib v1.5.0
```

**v2+ — 主版本升级**：import path 必须包含 `/v2`
```go
require github.com/example/lib/v2 v2.0.0
import "github.com/example/lib/v2/pkg"
```

这允许同一项目同时依赖 v1 和 v2：
```go
import (
    v1 "github.com/example/lib/pkg"
    v2 "github.com/example/lib/v2/pkg"
)
```

---

## 2 构建时注入版本信息（ldflags）

### 2.1 原理

Go 链接器支持在编译时设置包级别 `string` 变量的值：

```bash
go build -ldflags="-X 'main.version=v1.2.3'" -o myapp .
```

### 2.2 标准三元组：version + commit + buildTime

```go
package main

import (
    "flag"
    "fmt"
    "os"
)

var (
    version   = "dev"
    commit    = "none"
    buildTime = "unknown"
)

func main() {
    showVersion := flag.Bool("version", false, "显示版本信息")
    flag.Parse()

    if *showVersion {
        fmt.Printf("myapp %s (commit: %s, built: %s)\n",
            version, commit, buildTime)
        os.Exit(0)
    }
    fmt.Println("Application started")
}
```

构建脚本：

```bash
VERSION=$(git describe --tags --always --dirty)
COMMIT=$(git rev-parse --short HEAD)
BUILD_TIME=$(date -u '+%Y-%m-%dT%H:%M:%SZ')

go build -ldflags="\
  -X 'main.version=${VERSION}' \
  -X 'main.commit=${COMMIT}' \
  -X 'main.buildTime=${BUILD_TIME}' \
  -s -w" \
  -o myapp .
```

> **反例**: [trap/no-version-info/](trap/no-version-info/) — 部署后无法确定运行版本
>
> **反例**: [trap/hardcoded-version/](trap/hardcoded-version/) — 手动维护版本号导致不同步

---

## 3 goreleaser 详解

### 3.1 .goreleaser.yml 配置

```yaml
version: 2

before:
  hooks:
    - go mod tidy
    - go test ./...

builds:
  - id: myapp
    main: ./cmd/myapp
    binary: myapp
    env:
      - CGO_ENABLED=0
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.buildTime={{.Date}}

archives:
  - format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip

checksum:
  name_template: "checksums.txt"

changelog:
  sort: asc
  groups:
    - title: "New Features"
      regexp: '^.*?feat(\([[:word:]]+\))??!?:.+$'
    - title: "Bug Fixes"
      regexp: '^.*?fix(\([[:word:]]+\))??!?:.+$'
    - title: "Performance"
      regexp: '^.*?perf(\([[:word:]]+\))??!?:.+$'
```

### 3.2 Docker 镜像发布

```yaml
dockers:
  - image_templates:
      - "ghcr.io/myorg/myapp:{{ .Version }}"
      - "ghcr.io/myorg/myapp:latest"
    dockerfile: Dockerfile
    build_flag_templates:
      - "--label=org.opencontainers.image.revision={{.FullCommit}}"
```

### 3.3 Homebrew tap

```yaml
brews:
  - repository:
      owner: myorg
      name: homebrew-tap
    homepage: "https://github.com/myorg/myapp"
    description: "My CLI tool"
    install: |
      bin.install "myapp"
```

### 3.4 与 GitHub Actions 集成

```yaml
# .github/workflows/release.yml
name: Release
on:
  push:
    tags: ["v*"]

permissions:
  contents: write
  packages: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0       # goreleaser 需要完整 git 历史
      - uses: actions/setup-go@v5
        with:
          go-version: stable
      - uses: goreleaser/goreleaser-action@v6
        with:
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

> **反例**: [trap/manual-release/](trap/manual-release/) — 手动发布容易遗漏步骤

---

## 4 配置管理

### 4.1 viper 使用

```go
import "github.com/spf13/viper"

viper.SetDefault("server.port", 8080)
viper.SetConfigName("config")
viper.SetConfigType("yaml")
viper.AddConfigPath(".")
viper.SetEnvPrefix("APP")
viper.AutomaticEnv()
viper.ReadInConfig()
```

### 4.2 配置优先级

**正确顺序**（符合 12-Factor 原则）：

```
命令行 flag（最高）
    ↓
环境变量
    ↓
配置文件
    ↓
默认值（最低）
```

> **反例**: [trap/config-priority-wrong/](trap/config-priority-wrong/) — 优先级反转导致环境变量被配置文件覆盖

### 4.3 12-Factor App 配置原则

- 配置与代码严格分离
- 敏感信息**必须**用环境变量
- 启动时验证必要配置

```go
func validateConfig() error {
    required := []string{"DATABASE_URL", "REDIS_URL", "JWT_SECRET"}
    for _, key := range required {
        if os.Getenv(key) == "" {
            return fmt.Errorf("必需的环境变量 %s 未设置", key)
        }
    }
    return nil
}
```

### 4.4 配置文件格式选择

| 格式 | 优点 | 缺点 | 推荐场景 |
|------|------|------|---------|
| YAML | 可读性好，支持注释 | 缩进敏感 | K8s、通用配置 |
| TOML | 语义明确 | 嵌套深时冗长 | Go 项目 |
| JSON | 通用性最强 | 不支持注释 | API 配置 |
| ENV | 最简单 | 不支持嵌套 | Docker、12-Factor |

> **性能对比**: [performance/config-parse/](performance/config-parse/) — 不同格式的解析性能

### 4.5 环境变量命名约定

```bash
APP_SERVER_PORT=8080        # 应用前缀_模块_配置项
APP_DB_HOST=localhost
APP_DB_PASSWORD=secret      # 敏感信息只通过环境变量
APP_LOG_LEVEL=info
```

---

## 5 发布流程自动化

### 5.1 标准流水线

```
tag ──→ CI ──→ test ──→ build ──→ release ──→ notify
 │       │       │        │          │          │
 │       │       │        │          │          └─ Slack/飞书
 │       │       │        │          └─ GitHub Release + Docker
 │       │       │        └─ goreleaser 多平台
 │       │       └─ go test -race ./...
 │       └─ GitHub Actions
 └─ git tag -a v1.2.3 -m "Release v1.2.3"
```

### 5.2 发布检查清单

```markdown
发布前（自动化）：
- [ ] go test -race ./...
- [ ] golangci-lint run
- [ ] gosec ./...
- [ ] govulncheck ./...

发布后（验证）：
- [ ] myapp --version 显示正确
- [ ] Docker 镜像可拉取
- [ ] 冒烟测试通过
```

---

## 6 Go module 发布

### 6.1 GOPROXY 配置

```bash
# 默认
GOPROXY=https://proxy.golang.org,direct

# 国内推荐
GOPROXY=https://goproxy.cn,direct

# 私有仓库绕过 proxy
GOPRIVATE=github.com/mycompany/*
```

### 6.2 retract 指令

标记有问题的版本不应被使用：

```go
// go.mod
module github.com/example/lib

retract (
    v1.0.0           // 包含安全漏洞
    [v1.1.0, v1.1.3] // 数据库迁移 bug
)
```

发布 retract 的流程：
1. 在 go.mod 中添加 retract 指令
2. 发布新的补丁版本（如 v1.0.1）
3. 用户 `go get -u` 时会跳过被 retract 的版本

**注意**：Go module proxy 缓存意味着已发布的版本无法真正删除，retract 是正确的做法。

---

## 总结

| 主题 | 关键实践 |
|------|---------+
| 版本号 | 严格遵循 SemVer，v2+ 修改 import path |
| 版本注入 | ldflags 注入 version/commit/buildTime |
| 自动发布 | goreleaser + GitHub Actions |
| 配置管理 | flag > env > config > default |
| Module 发布 | 善用 GOPROXY 和 retract |

**常见陷阱**：
- 不注入版本信息：[trap/no-version-info/](trap/no-version-info/)
- 硬编码版本号：[trap/hardcoded-version/](trap/hardcoded-version/)
- 配置优先级错误：[trap/config-priority-wrong/](trap/config-priority-wrong/)
- 手动发布流程：[trap/manual-release/](trap/manual-release/)

**性能对比**：
- 配置格式解析：[performance/config-parse/](performance/config-parse/)
- ldflags vs go:embed：[performance/ldflags-vs-embed/](performance/ldflags-vs-embed/)