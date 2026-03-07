
---
Go项目布局详解
---

Go没有像Java/Maven那样的强制项目结构标准，但社区在多年实践中形成了一套事实上的约定。
理解这些约定，能让你的项目对其他Go开发者**立即可读**，也让工具链（`go build`、`go install`、`go test`）
与项目结构天然契合。

本文从Go语言**编译器和模块系统的设计约束**出发，解释每个目录约定背后的"为什么"，
而不仅仅是"是什么"。


# 1 Go项目的三种形态

在讨论目录结构前，先区分三种常见的Go项目形态，因为它们的布局需求差异很大：

| 形态 | 特征 | 典型代表 |
|------|------|---------|
| 可执行程序 | 有`main`包，产出二进制 | API服务、CLI工具、微服务 |
| 库（library） | 无`main`包，被其他项目import | `golang.org/x/sync`、`go-redis` |
| 混合型 | 既是库，又提供CLI工具 | `cobra`（库+`cobra-cli`工具） |

**核心原则**：布局为项目形态服务，不要为了"标准"而标准。一个50行的CLI工具不需要cmd/internal/pkg三层目录。


# 2 核心目录约定

## 2.1 cmd/ — 可执行入口

`cmd/`下的每个子目录对应一个可执行程序，子目录名即二进制名：

```
cmd/
├── server/
│   └── main.go    → go build -o server ./cmd/server
├── worker/
│   └── main.go    → go build -o worker ./cmd/worker
└── cli/
    └── main.go    → go build -o cli ./cmd/cli
```

**规则**：
- 每个子目录是独立的`package main`
- main.go应该尽量**薄**——只做参数解析、依赖组装、启动，不含业务逻辑
- 实际逻辑放在`internal/`或根包中

```go
// cmd/server/main.go — 薄入口示例
package main

import (
    "log"
    "myapp/internal/server"
    "myapp/internal/config"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatal(err)
    }

    srv := server.New(cfg)
    if err := srv.Run(); err != nil {
        log.Fatal(err)
    }
}
```

**为什么需要cmd/?**

一个项目可能产出多个二进制。如果直接在根目录放main.go，那只能有一个程序。
`cmd/`让多个程序共享同一个模块的代码（`internal/`），同时各自有独立入口。

**何时不需要cmd/?**

如果项目只有一个可执行文件，main.go放在根目录完全可以：

```
myapp/
├── main.go          # 唯一入口，没必要套cmd/
├── handler.go
└── handler_test.go
```

## 2.2 internal/ — 访问控制屏障

`internal/`是Go编译器**强制执行**的访问控制——`internal/`下的包只能被`internal/`的**父目录**及其子树import：

```
myapp/
├── cmd/server/main.go     ✅ 可以import myapp/internal/...
├── internal/
│   ├── handler/           ✅ 可以import myapp/internal/service/
│   ├── service/
│   └── repo/
└── pkg/api/               ✅ 可以import myapp/internal/...（同一模块内）
```

但外部项目：

```go
import "myapp/internal/handler" // ❌ 编译错误！
// use of internal package myapp/internal/handler not allowed
```

这是**编译器级别**的封装，不是约定，不是lint规则，是**硬性限制**。

**internal/的组织方式**：

```
internal/
├── config/        # 配置加载
├── handler/       # HTTP/gRPC handler
├── service/       # 业务逻辑层
├── repo/          # 数据访问层
├── model/         # 领域模型（结构体定义）
└── middleware/     # 中间件
```

**为什么internal/如此重要?**

Go的包一旦被外部项目import，就成了**公开API**——你必须为它做兼容性维护。
`internal/`让你自由重构内部实现，不用担心破坏外部调用方。

> "Make the zero value useful. Make the internal package your friend."  
> — 社区经验

## 2.3 pkg/ — 可复用的公开库

`pkg/`存放**可以被外部项目import**的公开包。注意：`pkg/`只是一个约定，编译器不做特殊处理。

```
myapp/
├── pkg/
│   ├── httpclient/    # 封装的HTTP客户端
│   ├── validator/     # 通用校验工具
│   └── retry/         # 重试逻辑
```

外部项目可以import：

```go
import "myapp/pkg/retry"
```

**pkg/的争议**：

Go社区对`pkg/`目录的态度并不统一：

| 观点 | 理由 |
|------|------|
| 支持 | 与`internal/`形成清晰的"公开 vs 私有"对比 |
| 反对 | Go已经有大小写导出机制，`pkg/`是冗余的路径层级 |
| 折中 | 大项目用`pkg/`、小项目不需要 |

**实际做法**：
- **库项目**：直接在根目录组织包，不需要`pkg/`。例如`github.com/go-redis/redis`直接`import "github.com/go-redis/redis/v9"`
- **应用项目**：如果有需要暴露的通用工具包，放`pkg/`；否则全放`internal/`

```
# 库项目：根目录即包
go-redis/
├── redis.go         # package redis
├── commands.go
└── options.go

# 应用项目：internal为主，pkg可选
myservice/
├── cmd/server/
├── internal/        # 90%的代码在这里
└── pkg/sdk/         # 只有需要暴露给调用方的SDK
```


# 3 辅助目录约定

## 3.1 api/ — 接口定义

存放API定义文件（非Go代码）：

```
api/
├── openapi/
│   └── v1.yaml        # OpenAPI/Swagger定义
├── proto/
│   ├── user.proto      # gRPC/Protobuf定义
│   └── order.proto
└── graphql/
    └── schema.graphql  # GraphQL schema
```

## 3.2 configs/ — 配置模板

存放配置文件模板或默认配置（不包含敏感信息）：

```
configs/
├── config.yaml.example
├── docker-compose.yaml
└── nginx.conf.template
```

## 3.3 scripts/ — 脚本

构建、安装、分析等辅助脚本：

```
scripts/
├── build.sh
├── migrate.sh
└── generate.sh
```

## 3.4 deployments/ — 部署配置

```
deployments/
├── docker/
│   └── Dockerfile
├── k8s/
│   ├── deployment.yaml
│   └── service.yaml
└── terraform/
```

## 3.5 test/ — 外部测试

存放集成测试、E2E测试等不适合放在包内的测试数据和工具：

```
test/
├── testdata/          # 测试用的fixture数据
├── integration/       # 集成测试
└── e2e/              # 端到端测试
```

注意：Go的单元测试约定放在被测包内（`_test.go`），不放在`test/`目录。

## 3.6 docs/ — 文档

```
docs/
├── architecture.md
├── api-guide.md
└── images/
```

## 3.7 tools/ — 项目工具

项目开发中需要的工具代码（代码生成器、lint规则等）：

```
tools/
├── tools.go           # 工具依赖声明（见下文）
└── codegen/
    └── main.go
```

`tools.go`用于固定工具版本（"tool dependencies"技巧）：

```go
//go:build tools

package tools

import (
    _ "golang.org/x/tools/cmd/stringer"
    _ "github.com/golangci/golangci-lint/cmd/golangci-lint"
)
```

这样`go mod tidy`会将这些工具纳入`go.sum`管理版本。


# 4 完整布局示例

## 4.1 小型项目（CLI工具/简单服务）

```
mytool/
├── main.go              # 直接在根目录
├── app.go               # 核心逻辑
├── app_test.go
├── config.go
├── go.mod
├── go.sum
└── README.md
```

小项目不需要`cmd/internal/pkg`——**不要过度设计**。

## 4.2 中型项目（单体API服务）

```
myapi/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── handler/
│   │   ├── user.go
│   │   └── user_test.go
│   ├── service/
│   │   ├── user.go
│   │   └── user_test.go
│   ├── repo/
│   │   ├── user.go
│   │   └── user_test.go
│   ├── model/
│   │   └── user.go
│   └── middleware/
│       ├── auth.go
│       └── logging.go
├── migrations/
│   ├── 001_create_users.up.sql
│   └── 001_create_users.down.sql
├── configs/
│   └── config.yaml.example
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## 4.3 大型项目（多服务 + SDK）

```
platform/
├── cmd/
│   ├── api-server/
│   │   └── main.go
│   ├── worker/
│   │   └── main.go
│   └── admin-cli/
│       └── main.go
├── internal/
│   ├── api/              # API服务逻辑
│   │   ├── handler/
│   │   ├── service/
│   │   └── repo/
│   ├── worker/           # Worker逻辑
│   │   ├── consumer/
│   │   └── processor/
│   ├── shared/           # 内部共享代码
│   │   ├── auth/
│   │   ├── cache/
│   │   └── database/
│   └── model/
├── pkg/
│   ├── sdk/              # 对外暴露的SDK
│   │   ├── client.go
│   │   └── client_test.go
│   └── errors/           # 公共错误类型
│       └── errors.go
├── api/
│   └── proto/
│       └── service.proto
├── deployments/
│   ├── docker/
│   └── k8s/
├── scripts/
│   ├── build.sh
│   └── generate.sh
├── go.mod
├── go.sum
├── Makefile
└── README.md
```


# 5 常见误区

## 5.1 过度设计

```
# 错误：50行的工具套了完整enterprise布局
tiny-cli/
├── cmd/
│   └── tiny-cli/
│       └── main.go      # 3行：调用internal
├── internal/
│   └── app/
│       └── app.go       # 10行：调用pkg
├── pkg/
│   └── core/
│       └── core.go      # 实际逻辑
```

**正确**：直接`main.go` + 一两个文件搞定。随着项目增长再重构。

## 5.2 照搬 golang-standards/project-layout

`github.com/golang-standards/project-layout` 这个仓库在社区**争议很大**。
Go团队成员Russ Cox曾明确表示这不是Go官方标准：

> "This is not a standard Go project layout."

该仓库的一些建议（如`pkg/`目录）在很多场景下是不必要的。建议把它作为参考之一，但不要盲目照搬。

## 5.3 src/目录

```
# 错误：Go不需要src/目录
myproject/
└── src/           # Java思维，Go不需要
    └── main.go
```

Go模块系统以`go.mod`为根，不需要也不应该有`src/`目录。这是从Java/C带过来的习惯。

## 5.4 按类型分包（models/controllers/utils）

```
# 不推荐：按类型分包
internal/
├── models/        # 所有模型
├── controllers/   # 所有handler
├── services/      # 所有service
└── utils/         # 万能工具箱

# 推荐：按领域/功能分包
internal/
├── user/          # 用户领域：model + service + repo
├── order/         # 订单领域
├── payment/       # 支付领域
└── auth/          # 认证模块
```

按类型分包会导致包之间**循环依赖**，按领域分包更符合Go的包设计哲学。

## 5.5 utils/common/shared包

```
# 不推荐：万能垃圾桶包
utils/
├── string_utils.go
├── time_utils.go
├── http_utils.go
└── ... 几十个不相关的函数
```

这类包违反了**高内聚**原则。正确做法是按功能拆分：`stringx/`、`httputil/`、`timeutil/`，
或者直接放到使用方的包中（如果只有一个调用者）。


# 6 Go工具链与目录的关系

## 6.1 go build

```bash
# 构建cmd/下的特定程序
go build -o bin/server ./cmd/server

# 构建所有cmd
go build ./cmd/...
```

`go build`的输出二进制名默认取目录名，所以`cmd/server/main.go`构建出`server`。

## 6.2 go test

```bash
# 测试整个项目
go test ./...

# 测试特定包
go test ./internal/service/...

# internal/的测试正常运行，不受访问限制（同模块内）
```

## 6.3 go install

```bash
# 安装到$GOPATH/bin（或$GOBIN）
go install ./cmd/server

# 外部用户安装你的工具
go install github.com/you/project/cmd/mytool@latest
```

`go install`要求目标是`main`包。`cmd/`下的每个子目录都可以独立安装。

## 6.4 go generate

```bash
# 在项目根目录运行所有generate指令
go generate ./...
```

`//go:generate`指令可以放在任何`.go`文件中，工具生成的代码建议放在同一个包内，
文件名加`_gen.go`或`_generated.go`后缀（gitignore可选）。


# 7 分包原则

目录约定解决的是顶层结构问题，但`internal/`内部如何组织同样重要。

## 7.1 包的命名

```go
// 好：简短、小写、无下划线
package user
package httputil
package auth

// 不好
package user_service    // 不要下划线
package userService     // 不要驼峰
package base            // 太泛
package util            // 太泛
```

## 7.2 包的大小

一个包的规模应该**足够小以便理解，足够大以保持独立**：

| 信号 | 可能需要拆分 | 可能需要合并 |
|------|------------|------------|
| 文件数 | >15个文件 | 1个文件+10行 |
| 导出符号 | >30个导出类型/函数 | 只有1-2个 |
| 依赖 | import了10+个同项目的包 | 无人import |
| 职责 | 名字需要用And/Or描述 | 功能完全被另一个包包含 |

## 7.3 依赖方向

```
cmd/ → internal/ → model/（无逆向依赖）

handler → service → repo → model
   ↓         ↓        ↓
  接口定义在使用方（见 interface/接口详解.md §8.3）
```

依赖应该是**单向**的、**从上到下**的。如果出现循环依赖，说明包的边界划分有问题。
解决方案：
- 提取公共类型到独立包（如`model/`）
- 使用接口解耦（在消费侧定义接口）
- 合并过于细碎的包


# 8 总结

| 目录 | 作用 | 编译器强制？ | 何时需要 |
|------|------|------------|---------|
| `cmd/` | 可执行程序入口 | 否（约定） | 多个二进制时 |
| `internal/` | 私有代码，外部不可import | **是** | 几乎所有项目 |
| `pkg/` | 公开可复用的库代码 | 否（约定） | 需要暴露SDK时 |
| `api/` | 接口定义文件 | 否 | 有API规范时 |
| `configs/` | 配置模板 | 否 | 有配置文件时 |
| `scripts/` | 辅助脚本 | 否 | 有构建/部署脚本时 |
| `test/` | 外部测试数据 | 否 | 有集成/E2E测试时 |
| `tools/` | 开发工具依赖 | 否 | 有代码生成等工具时 |

**核心记忆点**：
1. **小项目不需要复杂布局**——main.go在根目录完全可以
2. **internal/是Go唯一的编译器级访问控制**——善用它保护实现细节
3. **按领域分包，不按类型分包**——避免循环依赖
4. **避免utils/common包**——按功能拆分，保持高内聚
5. **布局随项目增长演进**——先简单，需要时再重构
