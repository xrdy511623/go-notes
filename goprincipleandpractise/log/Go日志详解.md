
---
Go日志详解
---

日志是程序与运维之间的**唯一通信渠道**——程序运行时发生了什么、出了什么问题、性能瓶颈在哪里，
全靠日志传递。Go的日志生态经历了三个阶段：标准库`log`（Go 1.0）→ 社区方案（zap/zerolog）→
标准库`log/slog`（Go 1.21）。本文从原理到实战，系统讲清楚Go日志的方方面面。


# 1 标准库log：起点与局限

## 1.1 基本使用

Go 1.0自带的`log`包极其简单——全局logger、固定格式、写到`os.Stderr`：

```go
import "log"

func main() {
    log.Println("server starting")          // 2024/01/15 10:30:00 server starting
    log.Printf("port=%d", 8080)             // 2024/01/15 10:30:00 port=8080
    log.Fatal("cannot bind port")           // 打印后 os.Exit(1)
    log.Panic("unexpected state")           // 打印后 panic
}
```

自定义logger：

```go
logger := log.New(os.Stdout, "[APP] ", log.Ldate|log.Ltime|log.Lshortfile)
logger.Println("hello")
// [APP] 2024/01/15 10:30:00 main.go:12: hello
```

## 1.2 log包的Flag常量

```go
const (
    Ldate         = 1 << iota  // 日期：2009/01/23
    Ltime                      // 时间：01:23:23
    Lmicroseconds              // 微秒精度：01:23:23.123123
    Llongfile                  // 完整文件路径+行号
    Lshortfile                 // 文件名+行号
    LUTC                       // 使用UTC时间
    Lmsgprefix                 // prefix放在消息前而非行首（Go 1.14+）
    LstdFlags     = Ldate | Ltime  // 默认
)
```

## 1.3 log包的局限

标准库`log`在生产环境有几个致命缺陷：

| 缺陷 | 说明 |
|------|------|
| 无日志级别 | 没有DEBUG/INFO/WARN/ERROR区分，无法按级别过滤 |
| 纯文本格式 | 无法输出JSON，不利于日志采集（ELK、Loki等） |
| 无结构化字段 | 只能拼字符串，无法按字段查询 |
| 全局状态 | `log.SetOutput()`影响全局，库代码可能互相干扰 |
| 性能一般 | 每次调用都做格式化+锁+写入，高吞吐场景成瓶颈 |

这些缺陷催生了社区日志库的繁荣。


# 2 社区方案：zap与zerolog

## 2.1 zap

Uber开源的[zap](https://github.com/uber-go/zap)是Go社区最流行的高性能日志库。

**核心设计**：两层API——`SugaredLogger`（方便）和`Logger`（零分配，极致性能）。

```go
import "go.uber.org/zap"

// 生产环境：JSON格式、Info级别、采样
logger, _ := zap.NewProduction()
defer logger.Sync()

// 结构化日志：零分配的强类型字段
logger.Info("user login",
    zap.String("username", "alice"),
    zap.Int("attempt", 3),
    zap.Duration("latency", 230*time.Millisecond),
)
// {"level":"info","ts":1705312200,"msg":"user login","username":"alice","attempt":3,"latency":"230ms"}

// Sugar模式：更方便，但有少量分配
sugar := logger.Sugar()
sugar.Infow("user login",
    "username", "alice",
    "attempt", 3,
)
sugar.Infof("server started on port %d", 8080)
```

**zap的零分配技巧**：
- `zap.String()`等函数返回`Field`结构体（值类型，栈分配）
- 使用`sync.Pool`复用`Buffer`
- 编码时直接写入`[]byte`，不经过`fmt.Sprintf`

**zap层级**（从低到高）：`Debug < Info < Warn < Error < DPanic < Panic < Fatal`

**常用配置**：

```go
cfg := zap.Config{
    Level:            zap.NewAtomicLevelAt(zap.InfoLevel),
    Encoding:         "json",         // 或 "console"
    OutputPaths:      []string{"stdout", "/var/log/app.log"},
    ErrorOutputPaths: []string{"stderr"},
    EncoderConfig:    zap.NewProductionEncoderConfig(),
}
logger, _ := cfg.Build()
```

**运行时动态调级**：

```go
lvl := zap.NewAtomicLevelAt(zap.InfoLevel)
logger, _ := zap.NewProduction(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
    return zapcore.NewCore(
        zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
        zapcore.AddSync(os.Stdout),
        lvl,
    )
}))

// 运行时通过HTTP接口调整级别（无需重启）
http.Handle("/log/level", lvl) // PUT /log/level -d '{"level":"debug"}'
```

## 2.2 zerolog

[zerolog](https://github.com/rs/zerolog)的设计目标是**零分配JSON日志**：

```go
import "github.com/rs/zerolog/log"

log.Info().
    Str("username", "alice").
    Int("attempt", 3).
    Dur("latency", 230*time.Millisecond).
    Msg("user login")
// {"level":"info","username":"alice","attempt":3,"latency":230,"message":"user login"}
```

**zerolog vs zap**：

| 维度 | zap | zerolog |
|------|-----|---------|
| API风格 | 函数参数`zap.String(k,v)` | 链式调用`.Str(k,v)` |
| 零分配 | Logger模式零分配 | 几乎所有操作零分配 |
| 性能 | 极高 | 略快于zap（链式避免了slice分配） |
| JSON以外 | 支持console/自定义 | 主要面向JSON |
| 生态 | 更大（gin-zap、grpc-zap等） | 较小但够用 |
| 动态调级 | 原生支持（AtomicLevel） | 需要手动实现 |

**选择建议**：两者都是生产级选择。zap生态更大、文档更全；zerolog API更简洁、性能略好。


# 3 log/slog：标准库的终极方案（Go 1.21+）

## 3.1 为什么需要slog

Go 1.21之前，每个项目都要做一个痛苦的选择：用哪个日志库？而且不同库的logger无法互通——
你的代码用zap，依赖的库用logrus，输出格式完全不统一。

`log/slog`的目标是提供一个**标准化的结构化日志API**，让整个生态统一：

```
应用代码 → slog.Logger → Handler接口 → 任何后端
                                          ├── slog.TextHandler（文本）
                                          ├── slog.JSONHandler（JSON）
                                          ├── zapslog（桥接zap）
                                          └── 自定义Handler
```

## 3.2 基本使用

```go
import "log/slog"

// 默认logger（文本格式，输出到stderr）
slog.Info("server starting", "port", 8080)
// time=2024-01-15T10:30:00.000+08:00 level=INFO msg="server starting" port=8080

// JSON格式
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
logger.Info("user login",
    "username", "alice",
    "attempt", 3,
)
// {"time":"2024-01-15T10:30:00.000+08:00","level":"INFO","msg":"user login","username":"alice","attempt":3}
```

## 3.3 四种日志级别

slog定义了4个内置级别（对应整数值，可自定义中间级别）：

```go
const (
    LevelDebug Level = -4
    LevelInfo  Level = 0
    LevelWarn  Level = 4
    LevelError Level = 8
)
```

级别间隔为4，是为了允许插入自定义级别（如`LevelNotice = 2`）。

```go
// 设置最低级别
handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
})
logger := slog.New(handler)

logger.Debug("detailed info")  // 会输出
logger.Info("normal info")
logger.Warn("warning")
logger.Error("error occurred")
```

## 3.4 结构化字段

slog支持两种写法——交替key-value和强类型Attr：

```go
// 写法1：交替key-value（方便，有少量分配）
slog.Info("request",
    "method", "GET",
    "path", "/api/users",
    "status", 200,
    "latency", 23*time.Millisecond,
)

// 写法2：强类型Attr（无分配，性能更好）
slog.LogAttrs(context.Background(), slog.LevelInfo, "request",
    slog.String("method", "GET"),
    slog.String("path", "/api/users"),
    slog.Int("status", 200),
    slog.Duration("latency", 23*time.Millisecond),
)
```

## 3.5 分组（Group）

Group将相关字段归入一个命名空间，输出时体现为嵌套：

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

logger.Info("request completed",
    slog.Group("request",
        slog.String("method", "GET"),
        slog.String("path", "/api/users"),
    ),
    slog.Group("response",
        slog.Int("status", 200),
        slog.Duration("latency", 23*time.Millisecond),
    ),
)
// {"time":"...","level":"INFO","msg":"request completed",
//  "request":{"method":"GET","path":"/api/users"},
//  "response":{"status":200,"latency":"23ms"}}
```

## 3.6 With：携带固定字段

`With`创建一个携带预设字段的子logger，后续所有日志自动附带这些字段：

```go
// 为每个请求创建带request_id的子logger
func handleRequest(w http.ResponseWriter, r *http.Request) {
    requestID := r.Header.Get("X-Request-ID")
    logger := slog.With("request_id", requestID, "method", r.Method)

    logger.Info("request started")
    // ... 处理逻辑 ...
    logger.Info("request completed", "status", 200)
}
// 两条日志都会自动带上 request_id 和 method
```

## 3.7 WithGroup

`WithGroup`创建一个子logger，后续所有字段都归入该group：

```go
dbLogger := slog.Default().WithGroup("db")
dbLogger.Info("query executed",
    "table", "users",
    "rows", 42,
)
// time=... level=INFO msg="query executed" db.table=users db.rows=42
```

## 3.8 context集成

slog原生支持从context传递logger，适合在请求链路中透传：

```go
// 中间件：将logger注入context
func LoggerMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        logger := slog.With(
            "request_id", r.Header.Get("X-Request-ID"),
            "remote_addr", r.RemoteAddr,
        )
        ctx := context.WithValue(r.Context(), loggerKey, logger)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// 业务代码：从context取logger
func handleOrder(ctx context.Context, orderID string) {
    logger := ctx.Value(loggerKey).(*slog.Logger)
    logger.InfoContext(ctx, "processing order", "order_id", orderID)
}
```

slog的`InfoContext`、`WarnContext`等方法接受context参数，方便Handler实现提取
context中的trace_id等信息。

## 3.9 Handler接口

slog的核心抽象是`Handler`接口：

```go
type Handler interface {
    Enabled(context.Context, Level) bool        // 该级别是否启用
    Handle(context.Context, Record) error       // 处理一条日志记录
    WithAttrs(attrs []Attr) Handler             // 添加预设字段
    WithGroup(name string) Handler              // 添加分组
}
```

**内置Handler**：
- `slog.TextHandler`：人类可读的`key=value`格式
- `slog.JSONHandler`：机器可读的JSON格式

**自定义Handler示例**：给日志加上颜色（开发环境）

```go
type ColorHandler struct {
    slog.Handler
    w io.Writer
}

func (h *ColorHandler) Handle(ctx context.Context, r slog.Record) error {
    var color string
    switch {
    case r.Level >= slog.LevelError:
        color = "\033[31m" // 红色
    case r.Level >= slog.LevelWarn:
        color = "\033[33m" // 黄色
    default:
        color = "\033[0m"  // 默认
    }
    fmt.Fprintf(h.w, "%s[%s] %s%s\n", color, r.Level, r.Message, "\033[0m")
    return nil
}
```

## 3.10 slog与zap桥接

如果项目已经使用zap，可以通过`zapslog`适配器让slog和zap共存：

```go
import "go.uber.org/zap/exp/zapslog"

zapLogger, _ := zap.NewProduction()
slogHandler := zapslog.NewHandler(zapLogger.Core(), nil)
slog.SetDefault(slog.New(slogHandler))

// 现在slog的输出会经过zap的Core处理
slog.Info("hello from slog") // 由zap格式化输出
```


# 4 日志级别策略

## 4.1 各级别使用指南

| 级别 | 使用场景 | 生产环境是否开启 |
|------|---------|----------------|
| DEBUG | 开发调试信息、变量值、SQL语句 | 关闭（按需临时开启） |
| INFO | 业务流程关键节点、启动/关闭信息 | 开启 |
| WARN | 可恢复的异常、即将过期的配置 | 开启 |
| ERROR | 不可恢复的错误、需要人工介入 | 开启 |

## 4.2 该用哪个级别？

```
用户发送请求          → INFO （业务事件）
请求参数校验失败       → WARN （客户端错误，不需要报警）
数据库查询成功         → DEBUG（常规操作，量太大）
数据库连接失败         → ERROR（需要报警）
服务启动成功          → INFO
配置项即将过期         → WARN
外部API返回500       → ERROR
缓存未命中，回源成功    → DEBUG
缓存未命中率超过阈值    → WARN
```

## 4.3 动态调级

生产环境通常设置为INFO级别，但排查问题时需要临时开启DEBUG。三种实现方式：

```go
// 方式1：slog + atomic.Value
var logLevel = new(slog.LevelVar)
handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})
slog.SetDefault(slog.New(handler))

// 运行时调整（可通过HTTP接口暴露）
logLevel.Set(slog.LevelDebug)

// 方式2：zap AtomicLevel（见2.1节）

// 方式3：环境变量（重启生效）
level := os.Getenv("LOG_LEVEL") // "debug", "info", "warn", "error"
```


# 5 生产环境最佳实践

## 5.1 日志输出格式

```
开发环境 → TextHandler / Console格式（人类可读）
生产环境 → JSONHandler / JSON格式（机器可读）
```

JSON日志是ELK（Elasticsearch + Logstash + Kibana）、Grafana Loki等日志系统的标准输入格式。

## 5.2 必要的日志字段

每条日志应包含足够的上下文：

```go
slog.Info("order created",
    "request_id", requestID,   // 请求追踪
    "user_id", userID,         // 谁触发的
    "order_id", orderID,       // 业务ID
    "amount", amount,          // 业务数据
    "latency_ms", latency.Milliseconds(), // 性能数据
)
```

推荐字段清单：

| 字段 | 说明 | 来源 |
|------|------|------|
| time | 时间戳 | slog自动添加 |
| level | 日志级别 | slog自动添加 |
| msg | 日志消息 | 必填 |
| request_id | 请求ID | HTTP Header / context |
| trace_id | 链路追踪ID | OpenTelemetry |
| user_id | 用户标识 | 认证中间件 |
| service | 服务名 | 配置/环境变量 |
| error | 错误信息 | 仅ERROR级别 |

## 5.3 日志与error的关系

**原则：记录error一次且仅一次**。

```go
// 错误：每层都记日志，同一个错误出现多次
func handler(w http.ResponseWriter, r *http.Request) {
    err := service.CreateOrder(ctx)
    if err != nil {
        slog.Error("handler: create order failed", "error", err)  // 第3次记录
        http.Error(w, "internal error", 500)
    }
}

func (s *Service) CreateOrder(ctx context.Context) error {
    err := s.repo.Insert(ctx, order)
    if err != nil {
        slog.Error("service: insert order failed", "error", err)  // 第2次记录
        return fmt.Errorf("create order: %w", err)
    }
    return nil
}

func (r *Repo) Insert(ctx context.Context, order Order) error {
    _, err := r.db.ExecContext(ctx, query, args...)
    if err != nil {
        slog.Error("repo: db exec failed", "error", err)  // 第1次记录
        return fmt.Errorf("insert order: %w", err)
    }
    return nil
}
```

```go
// 正确：底层只wrap error，顶层统一记录
func handler(w http.ResponseWriter, r *http.Request) {
    err := service.CreateOrder(ctx)
    if err != nil {
        slog.Error("create order failed", "error", err)  // 唯一的日志点
        http.Error(w, "internal error", 500)
    }
}

func (s *Service) CreateOrder(ctx context.Context) error {
    err := s.repo.Insert(ctx, order)
    if err != nil {
        return fmt.Errorf("create order: %w", err)  // 只wrap，不记日志
    }
    return nil
}

func (r *Repo) Insert(ctx context.Context, order Order) error {
    _, err := r.db.ExecContext(ctx, query, args...)
    if err != nil {
        return fmt.Errorf("insert order: %w", err)  // 只wrap，不记日志
    }
    return nil
}
```

详见 [错误处理](../handle-error/错误处理.md) 第5节最佳实践。

## 5.4 敏感信息脱敏

**绝对不要**在日志中输出以下信息：

```go
// 错误：记录了密码和token
slog.Info("user auth",
    "password", req.Password,       // ❌ 密码
    "token", req.Token,             // ❌ 认证令牌
    "credit_card", req.CardNumber,  // ❌ 信用卡号
)

// 正确：脱敏处理
slog.Info("user auth",
    "username", req.Username,
    "has_token", req.Token != "",    // 只记录是否有token
)
```

可以通过自定义`slog.Handler`统一拦截敏感字段：

```go
func (h *SanitizeHandler) Handle(ctx context.Context, r slog.Record) error {
    sanitized := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
    r.Attrs(func(a slog.Attr) bool {
        if isSensitive(a.Key) {
            sanitized.AddAttrs(slog.String(a.Key, "***REDACTED***"))
        } else {
            sanitized.AddAttrs(a)
        }
        return true
    })
    return h.inner.Handle(ctx, sanitized)
}
```

## 5.5 日志采样

高流量服务每秒可能产生数万条日志，全量写入会浪费I/O和存储。zap内置了采样机制：

```go
// zap：每秒第1条和之后每100条同类日志才输出
cfg := zap.NewProductionConfig()
cfg.Sampling = &zap.SamplingConfig{
    Initial:    1,   // 每秒第1条
    Thereafter: 100, // 之后每100条
}
```

slog没有内置采样，需要在自定义Handler中实现。

## 5.6 避免在热路径中记日志

```go
// 不好：循环内每次迭代都记日志
for _, item := range items {
    slog.Debug("processing item", "id", item.ID)
    process(item)
}

// 好：只记录汇总
slog.Info("processing batch", "count", len(items))
for _, item := range items {
    process(item)
}
slog.Info("batch completed", "count", len(items))
```


# 6 slog性能剖析

## 6.1 为什么slog比log快

| 机制 | log | slog |
|------|-----|------|
| 字段编码 | fmt.Sprintf拼字符串 | 直接写入[]byte |
| 内存分配 | 每次调用分配 | Attr值类型+池化Buffer |
| 级别检查 | 无（全部输出） | Enabled()前置检查 |
| 锁竞争 | 全局锁 | Handler自行管理 |

## 6.2 slog vs zap vs zerolog

三者都是高性能日志库，slog的优势在于**标准库身份**（零外部依赖、生态统一），
而zap/zerolog在极致性能场景有微弱优势。

详细性能对比见 [performance/log_bench_test.go](performance/log_bench_test.go)。

关键结论：
- 在绝大多数应用中，slog的性能**完全够用**
- 如果每秒百万条日志，zap/zerolog有5-20%的优势
- 选择slog的最大理由不是性能，而是**标准化和零依赖**


# 7 常见陷阱

## 7.1 log.Fatal在goroutine中

`log.Fatal`调用`os.Exit(1)`——它**不会执行defer**、**不会等其他goroutine结束**：

```go
// 危险：Fatal在goroutine中调用
go func() {
    if err := process(); err != nil {
        log.Fatal(err) // 整个进程直接退出！defer不执行！
    }
}()
```

正确做法：goroutine中用`slog.Error` + 返回错误，由主goroutine决定是否退出。

## 7.2 忘记logger.Sync()

zap的底层使用了缓冲写入，程序退出前必须刷新：

```go
logger, _ := zap.NewProduction()
defer logger.Sync() // 必须！否则最后几条日志可能丢失
```

slog的内置Handler没有缓冲，不需要Sync。但如果底层Writer有缓冲（如`bufio.Writer`），
同样需要在退出前Flush。

## 7.3 slog交替key-value奇数参数

```go
// 错误：key-value不成对，"alice"成了悬空的key
slog.Info("user login", "username", "alice", "attempt")
// time=... level=INFO msg="user login" username=alice !BADKEY=attempt

// 正确：确保key-value成对
slog.Info("user login", "username", "alice", "attempt", 3)
```

slog不会panic，但会输出`!BADKEY=`标记。用`go vet`可以检测到这类问题（Go 1.22+）。

## 7.4 日志中记录error但不处理

```go
// 错误：记了日志但吞掉了error，调用方不知道出错了
func doSomething() {
    if err := riskyOperation(); err != nil {
        slog.Error("operation failed", "error", err)
        // 没有return err！调用方以为成功了
    }
}
```

## 7.5 在init()中使用slog

```go
func init() {
    // 此时slog可能还没被配置（还是默认的TextHandler到stderr）
    slog.Info("package initialized")
}
```

`init()`执行时机在`main()`之前，如果`main()`中才配置slog，init中的日志会使用默认配置。

完整陷阱演示见 [trap/main.go](trap/main.go)。


# 8 总结

| 方案 | 推荐场景 | 日志级别 | 结构化 | 性能 |
|------|---------|---------|--------|------|
| log | 学习/脚本/快速原型 | 无 | 无 | 一般 |
| slog | **Go 1.21+新项目首选** | 4级(可扩展) | 原生支持 | 高 |
| zap | 超高性能/已有项目 | 7级 | 强类型Field | 极高 |
| zerolog | 超高性能/JSON优先 | 7级 | 链式API | 极高 |

**新项目技术选型建议**：

```
Go 1.21+ → 直接用slog
   │
   └─ 需要极致性能？→ slog + zap Handler桥接
   │
   └─ 需要特殊功能（采样/动态调级HTTP接口）？→ 直接用zap
```

**核心记忆点**：
- 生产环境用JSON格式，开发用Text格式
- 每条日志带 request_id 和关键业务字段
- error记录一次且仅一次——底层wrap，顶层记录
- 不在日志中输出密码、token等敏感信息
- goroutine中不用`log.Fatal`
- Go 1.21+新项目首选`slog`，旧项目可通过Handler桥接迁移
