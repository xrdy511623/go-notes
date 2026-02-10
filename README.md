# go-notes

`go-notes` 是一个偏向 **知识库 / 编程经验分享** 的开源仓库，核心目标是沉淀：
- Go 语言原理与工程实践
- 中间件（MySQL / Redis / Kafka）专题
- Linux 性能优化方法论
- 工程效率与工具使用经验


## 目录

- [项目定位](#项目定位)
- [内容总览](#内容总览)
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

| 模块 | Markdown | Go 文件 | 图片 |
| --- | ---: | ---: | ---: |
| `goprincipleandpractise` | 27 | 69 | 101 |
| `middlewares` | 32 | 0 | 186 |
| `designpattern` | 0 | 10 | 0 |
| `enhancelinuxperformance` | 35 | 0 | 102 |
| `productivetools` | 10 | 0 | 197 |
| `shellscripts` | 6 | 0 | 2 |
| `softskill` | 1 | 0 | 8 |

## 阅读路径建议

### 1) 如果你想系统补 Go 基础与进阶
优先阅读 `goprincipleandpractise/`：
- 并发：`channel`、`lock`、`sync`、`context`
- 数据结构与性能：`slice`、`map`、`string`、`struct`
- 工程能力：`unit-test`、`benchmark`、`pprof-practise`、`fuzzingtest`

### 2) 如果你在做后端基础设施
优先阅读 `middlewares/`：
- MySQL 专题（事务、锁、MVCC、索引、SQL 优化）
- Redis 专题（数据结构、持久化、主从、哨兵、集群）
- Kafka 入门与配置

### 3) 如果你在做线上性能治理
优先阅读 `enhancelinuxperformance/`：
- CPU / 内存 / IO / 网络四大类排障与优化
- 全链路观测工具与实战案例

### 4) 如果你关注个人工程效率
优先阅读 `productivetools/`：
- Git / Vim / IDE / 终端配置
- 搜索效率与 AI 工具实践

## 知识地图

### Go 原理与实践
- `goprincipleandpractise/channel/channel详解.md`
- `goprincipleandpractise/map/map详解.md`
- `goprincipleandpractise/slice/切片详解.md`
- `goprincipleandpractise/string/字符串拼接背后的原理.md`
- `goprincipleandpractise/lock/go语言中的锁详解.md`
- `goprincipleandpractise/sync/errgroup源码分析.md`

### 中间件
- `middlewares/mysql/`（18 篇）
- `middlewares/redis/`（12 篇）
- `middlewares/kafka/`（2 篇）

### Linux 性能
- `enhancelinuxperformance/`（01~35 系列）

### 设计模式与工程工具
- `designpattern/`（示例代码）
- `productivetools/`（效率工具与实践）

### Shell 与软技能
- `shellscripts/`
- `softskill/document-writing-practise/`

## 仓库结构

```text
go-notes/
├── goprincipleandpractise/             # Go 原理、性能优化、源码分析、踩坑
│   ├── benchmark/
│   ├── channel/
│   ├── gc/
│   ├── lock/
│   ├── map/
│   ├── slice/
│   ├── string/
│   ├── sync/
│   └── ...
├── middlewares/                        # MySQL / Redis / Kafka 专题
├── enhancelinuxperformance/            # Linux 性能优化 35 篇系列
├── designpattern/                      # 设计模式示例代码
├── productivetools/                    # Git/Vim/IDE/终端/AI 工具实践
├── shellscripts/                       # Shell 基础与脚本实践
├── softskill/                          # 软技能（技术写作）
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

- 文档 Owner：john.q

维护策略：
1. 代码示例变化时同步更新相关文档和图片。
2. 定期复查历史文档中的版本相关结论（Go / MySQL / Redis）。
3. 对高频访问专题优先补充“反例 + 边界条件 + 验证命令”。

## License

本项目采用 MIT License，见 `LICENSE`。

## 联系方式

- desperateslope@gmail.com
