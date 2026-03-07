# go-notes

`go-notes` 是一个偏向 **知识库 / 编程经验分享** 的开源仓库，核心目标是沉淀：
- Go 语言原理与工程实践
- 中间件（MySQL / Redis / Kafka）专题
- Linux 性能优化方法论
- 工程效率与工具使用经验


## 目录

- [项目定位](#项目定位)
- [内容总览](#内容总览)
- [快速开始](#快速开始)
- [阅读路径建议](#阅读路径建议)
- [知识地图](#知识地图)
- [仓库结构](#仓库结构)
- [内容组织约定](#内容组织约定)
- [如何贡献](#如何贡献)
- [维护与更新](#维护与更新)
- [License](#license)
- [联系方式](#联系方式)

## 项目定位

这是一个“文档 + 示例代码 + 图解证据”的技术笔记仓库，强调：
1. 用可运行的最小示例解释机制。
2. 用 benchmark / 测试结果验证结论。
3. 用图片和结构化文档提升可读性与复用性。

## 内容总览

基于仓库当前文件统计：

| 模块 | 定位 | Markdown | Go 文件 | 图片 |
| --- | --- | ---: | ---: | ---: |
| `gocore` | Go 语言原理与基础 | 30+ | 250+ | 80+ |
| `goengineering` | Go 工程化实践 | 15+ | 80+ | 40+ |
| `middlewares` | 中间件专题 | 33 | 0 | 194 |
| `designpattern` | 设计模式 | 26 | 134 | 0 |
| `linux-perf` | Linux 性能优化 | 35 | 0 | 102 |
| `productivetools` | 效率工具 | 11 | 0 | 197 |
| `shellscripts` | Shell 脚本 | 6 | 0 | 2 |
| `softskill` | 软技能 | 1 | 0 | 8 |

## 快速开始

### 运行环境

- Go: `1.24.0`（见 `go.mod`）

### 常用命令

```bash
# 安装依赖
go mod download

# 运行并发专题性能测试
go test ./gocore/channel/performance
go test ./gocore/context/performance

# 运行设计模式目录中的示例与测试
go run ./designpattern
go test ./designpattern/...

# 运行新增专题示例测试
go test ./gocore/concurrency/pattern \
  ./gocore/concurrency/performance \
  ./gocore/interface/performance \
  ./gocore/struct/performance/set

# 运行工程化实践相关测试
go test ./goengineering/unit-test/...
go test ./goengineering/benchmark/...
go test ./goengineering/api-design/...

# 运行 fuzzing 示例（当前可通过的子包）
go test ./goengineering/fuzzingtest/byteparser \
  ./goengineering/fuzzingtest/multiparam \
  ./goengineering/fuzzingtest/roundtrip \
  ./goengineering/fuzzingtest/differential

# 运行指定 Fuzz 目标（示例）
go test -run=^$ -fuzz=^FuzzParseAge$ -fuzztime=30s ./goengineering/fuzzingtest

# 复现 ParseAge 示例中的已知边界问题（预期失败）
go test -run=^FuzzParseAge$ ./goengineering/fuzzingtest
```

## 阅读路径建议

### 1) 如果你想系统补 Go 基础与进阶
优先阅读 `gocore/`：
- 并发：`channel`、`sync`、`context`、`concurrency`
- 数据结构与性能：`slice`、`map`、`string`、`struct`
- 运行时：`gc`、`gmp`、`generics`
- 网络与 IO：`net-http`、`websocket`、`database-sql`、`io`

### 2) 如果你想提升 Go 工程化能力
优先阅读 `goengineering/`：
- 测试：`unit-test`、`benchmark`、`fuzzingtest`、`integration-test`、`e2e-test`
- 构建与发布：`makefile`、`ci-cd`、`docker`、`release`
- 代码质量：`coding-standard`、`linter`、`secure-coding`、`codegen`
- 可观测性：`pprof-practise`、`trace`、`log`、`gops`
- 项目设计：`project-layout`、`api-design`、`manage-dependency`

### 3) 如果你在做后端基础设施
优先阅读 `middlewares/`：
- MySQL 专题（事务、锁、MVCC、索引、SQL 优化）
- Redis 专题（数据结构、持久化、主从、哨兵、集群）
- Kafka 入门与配置

### 4) 如果你在做线上性能治理
优先阅读 `linux-perf/`：
- CPU / 内存 / IO / 网络四大类排障与优化
- 全链路观测工具与实战案例

### 5) 如果你关注个人工程效率
优先阅读 `productivetools/`：
- Git / Vim / IDE / 终端配置
- 搜索效率与 AI 工具实践

## 知识地图

### Go 原理与实践
- `gocore/channel/channel详解.md`
- `gocore/map/map详解.md`
- `gocore/slice/切片详解.md`
- `gocore/string/详解go语言中的string.md`
- `gocore/lock/go语言中的锁详解.md`
- `gocore/context/context详解.md`
- `gocore/concurrency/并发进阶.md`
- `gocore/interface/接口详解.md`
- `gocore/sync/errgroup源码分析.md`

### Go 工程化实践
- `goengineering/unit-test/单元测试详解.md`
- `goengineering/benchmark/benchmark性能基准测试详解.md`
- `goengineering/fuzzingtest/详解go语言中的fuzzing.md`
- `goengineering/codegen/Go代码生成详解.md`
- `goengineering/log/Go日志详解.md`
- `goengineering/api-design/API设计规范.md`
- `goengineering/pprof-practise/` — 性能分析实战
- `goengineering/makefile/` — Makefile 设计与构建自动化
- `goengineering/ci-cd/` — CI/CD 与 PR 流程
- `goengineering/docker/` — Docker 化构建与镜像优化
- `goengineering/release/` — 版本管理与发布流程

### 中间件
- `middlewares/mysql/`（19 篇）
- `middlewares/redis/`（13 篇）
- `middlewares/kafka/`（2 篇）

### Linux 性能
- `linux-perf/`（01~35 系列）

### 设计模式
- `designpattern/README.md` — 25 种常用设计模式、并发模式与韧性模式总览
- `designpattern/` — 模式文档、可运行示例、`trap/` 反例与配套测试

### 效率工具与软技能
- `productivetools/` — Git / Vim / IDE / 终端 / AI 工具实践
- `shellscripts/` — Shell 基础与脚本实践
- `softskill/document-writing-practise/` — 技术写作

## 仓库结构

```text
go-notes/
├── gocore/                # Go 语言原理与基础
│   ├── channel/           #   channel 详解
│   ├── concurrency/       #   并发进阶
│   ├── context/           #   context 详解
│   ├── gc/                #   GC 与内存管理
│   ├── generics/          #   泛型
│   ├── gmp/               #   GMP 调度模型
│   ├── interface/         #   接口详解
│   ├── lock/              #   锁详解
│   ├── map/               #   map 详解
│   ├── slice/             #   切片详解
│   ├── string/            #   string 详解
│   ├── sync/              #   sync 包源码分析
│   └── ...                #   defer, for-range, select, io, net-http, ...
├── goengineering/         # Go 工程化实践
│   ├── api-design/        #   API 设计规范
│   ├── benchmark/         #   性能基准测试
│   ├── ci-cd/             #   CI/CD 与 PR 流程
│   ├── docker/            #   Docker 化构建
│   ├── makefile/          #   Makefile 设计
│   ├── pprof-practise/    #   性能分析实战
│   ├── project-layout/    #   项目布局
│   ├── release/           #   版本管理与发布
│   ├── unit-test/         #   单元测试详解
│   └── ...                #   codegen, linter, log, trace, ...
├── middlewares/            # MySQL / Redis / Kafka 专题
├── linux-perf/            # Linux 性能优化 35 篇系列
├── designpattern/         # 设计模式文档、示例代码与测试
├── productivetools/       # Git / Vim / IDE / 终端 / AI 工具实践
├── shellscripts/          # Shell 基础与脚本实践
├── softskill/             # 软技能（技术写作）
├── go.mod
├── go.sum
└── README.md
```

## 内容组织约定

仓库内常见组织模式：

1. `*.md`：原理讲解、经验总结、边界条件说明。  
2. `*.go` / `*_test.go`：最小可运行示例、测试或 benchmark。  
3. `images/`：图解与实验结果截图。  
4. `performance/`：性能对比实验。  
5. `trap/`：常见误区与反例。

## 如何贡献

欢迎提交 Issue / PR，建议遵循：

1. 结论优先证据化：
- 性能类结论附可复现步骤与关键输出。
- 原理类结论尽量给出代码或图解支撑。

2. 保持内容结构一致：
- 新专题优先沿用 `文档 + 示例 + 图片` 组织方式。
- 文件命名尽量语义化，便于检索。

3. 提交信息规范：
- 仓库存在 `.czrc`，配置为 `cz-conventional-changelog`。

## 维护与更新

维护策略：
1. 代码示例变化时同步更新相关文档和图片。
2. 定期复查历史文档中的版本相关结论（Go / MySQL / Redis）。
3. 对高频访问专题优先补充“反例 + 边界条件 + 验证命令”。

## License

本项目采用 MIT License，见 `LICENSE`。

## 联系方式

- desperateslope@gmail.com
