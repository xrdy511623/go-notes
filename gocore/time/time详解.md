# Go 标准库 time 详解

`time` 是 Go 标准库中处理时间的核心包。看似简单的时间操作背后隐藏着 wall clock 与 monotonic clock 的双时钟设计、定时器的运行时调度机制、以及令无数开发者困惑的时区陷阱。本文将从源码层面剖析其核心设计，覆盖 Timer/Ticker、时区处理、格式化等关键主题。

---

## 1 time.Time 的双时钟设计

### 1.1 wall clock vs monotonic clock

`time.Time` 内部同时保存了两个时钟读数：

```go
// 源码: time/time.go
type Time struct {
    wall uint64      // wall clock（日历时间）
    ext  int64       // monotonic clock（单调时钟）或扩展秒数
    loc  *Location   // 时区信息
}
```

**wall clock**（墙上时钟）：
- 对应操作系统的 `CLOCK_REALTIME`
- 可以被 NTP 校时、用户手动修改
- 用于"现在几点了"这类场景

**monotonic clock**（单调时钟）：
- 对应操作系统的 `CLOCK_MONOTONIC`
- 只增不减，不受 NTP 影响
- 用于"过了多久"这类场景

Go 1.9 开始，`time.Now()` 同时读取两个时钟：

```go
// 伪代码，简化自 runtime/time_now
func Now() Time {
    sec, nsec, mono := now() // 系统调用，同时获取两个时钟
    return Time{
        wall: hasMonotonic | sec<<nsecShift | nsec,
        ext:  mono,  // 保存 monotonic 读数
    }
}
```

### 1.2 为什么需要双时钟？

考虑一个超时判断：

```go
start := time.Now()
doSomething()
elapsed := time.Since(start) // time.Now().Sub(start)
```

如果在 `doSomething()` 执行期间，NTP 将系统时间往回调了 1 小时：
- **只用 wall clock**：`elapsed` 会是负数（-59 分 59 秒），逻辑崩溃
- **使用 monotonic clock**：`elapsed` 正确反映实际经过的时间

**规则：当两个 Time 都携带 monotonic 读数时，Sub/Before/After/Equal 使用 monotonic clock 比较；否则回退到 wall clock。**

```go
a := time.Now()                         // 携带 monotonic
b := time.Now()                         // 携带 monotonic
fmt.Println(b.Sub(a))                   // 使用 monotonic，不受 NTP 影响

c := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) // 无 monotonic
d := time.Now()                                     // 携带 monotonic
fmt.Println(d.Sub(c))                               // 回退到 wall clock
```

### 1.3 剥离 monotonic 读数

某些场景（如序列化、日志）只需要 wall clock：

```go
t := time.Now()
fmt.Println(t)              // "2024-01-15 10:30:00.123456 +0800 CST m=+0.001234567"
                            // 注意末尾的 m=+... 就是 monotonic 读数

t = t.Round(0)              // 剥离 monotonic 读数
fmt.Println(t)              // "2024-01-15 10:30:00.123456 +0800 CST"
// 或
t = t.Truncate(0)           // 同样效果
```

---

## 2 time.Duration

### 2.1 本质是纳秒计数

```go
// 源码: time/time.go
type Duration int64
```

`Duration` 就是一个 `int64`，表示纳秒数。预定义常量：

```go
const (
    Nanosecond  Duration = 1
    Microsecond          = 1000 * Nanosecond
    Millisecond          = 1000 * Microsecond
    Second               = 1000 * Millisecond
    Minute               = 60 * Second
    Hour                 = 60 * Minute
)
```

**int64 的极限**：`Duration` 最大约 290 年（`math.MaxInt64` 纳秒 ≈ 292 年）。对于绝大多数应用足够，但如果要表示地质年代级别的时长，需要自定义类型。

### 2.2 常见运算陷阱

```go
// 正确：常量乘法
timeout := 5 * time.Second           // Duration * int → Duration

// 错误：变量乘法需要类型转换
n := 5
timeout := time.Duration(n) * time.Second  // 正确
// timeout := n * time.Second              // 编译错误: mismatched types int and time.Duration

// 除法获取单位数
d := 2*time.Hour + 30*time.Minute
hours := d.Hours()       // float64: 2.5
minutes := d.Minutes()   // float64: 150
seconds := d.Seconds()   // float64: 9000

// 截断和取整
d = 1*time.Hour + 23*time.Minute + 45*time.Second
d = d.Truncate(time.Hour)   // 1h0m0s（向零截断）
d = d.Round(time.Hour)      // 1h0m0s（四舍五入）
```

### 2.3 字符串解析

```go
d, err := time.ParseDuration("1h30m10s")
if err != nil {
    log.Fatal(err)
}
fmt.Println(d) // 1h30m10s

// 支持的单位: ns, us/µs, ms, s, m, h
// 注意：不支持 "d"（天）和 "y"（年），因为天和年的长度不固定
```

---

## 3 Timer 与 Ticker

### 3.1 运行时定时器架构

Go 的定时器不是由 `time` 包自行管理的，而是深度集成在 **runtime** 中。Go 1.23 对定时器做了重大重构：

```
Go 1.23 之前:
  全局 4 个 timer 堆 (P 级别)
  每个 P 一个最小堆，插入/删除 O(log n)

Go 1.23+:
  timer 直接挂在 goroutine 的关联 P 上
  减少了锁竞争，提升了高并发场景下的性能
```

核心数据结构（简化）：

```go
// runtime/time.go
type timer struct {
    pp       puintptr  // 所属的 P
    when     int64     // 触发时间（monotonic clock）
    period   int64     // Ticker 的间隔（Timer 为 0）
    f        func(arg any, seq uintptr, delay int64) // 回调函数
    arg      any       // 回调参数
    seq      uintptr   // 序列号
}
```

### 3.2 time.NewTimer

```go
func NewTimer(d Duration) *Timer {
    t := &Timer{C: make(chan Time, 1)}  // 带缓冲的 channel
    t.init(sendTime, &t.C)
    t.Reset(d)
    return t
}
```

**关键设计：channel 缓冲区大小为 1**。这意味着：
- 定时器到期时，runtime 向 channel 发送一个值
- 如果 channel 已满（没人读），发送会被丢弃（不会阻塞 runtime）
- 这就是 `Reset` 前需要 drain channel 的原因

### 3.3 time.NewTicker

```go
func NewTicker(d Duration) *Ticker {
    if d <= 0 {
        panic("non-positive interval for NewTicker")
    }
    t := &Ticker{C: make(chan Time, 1)}
    t.init(sendTime, &t.C)
    t.Reset(d)
    return t
}
```

**Timer vs Ticker 的本质区别：**

| 特性 | Timer | Ticker |
|------|-------|--------|
| 触发次数 | 一次 | 重复 |
| runtime 字段 | `period = 0` | `period = d` |
| 零值 Duration | 允许 | panic |
| 必须 Stop | 到期后自动清理 | **必须 Stop，否则泄漏** |
| 典型场景 | 超时控制、延迟执行 | 定期轮询、心跳 |

### 3.4 Go 1.23 的重要变更

Go 1.23 对 Timer 和 Ticker 做了两个关键改变：

**1. Stop/Reset 后不再需要手动 drain channel：**

```go
// Go 1.23 之前，Reset 前需要 drain
if !timer.Stop() {
    <-timer.C  // 必须 drain，否则下次读取会立即返回旧值
}
timer.Reset(d)

// Go 1.23+，Stop 和 Reset 保证 channel 被清空
timer.Reset(d)  // 安全，无需手动 drain
```

**2. 未被引用的 Timer/Ticker 可以被 GC 回收（即使未 Stop）：**

```go
// Go 1.23 之前，这是内存泄漏
func handler() {
    time.NewTicker(time.Second) // 泄漏！即使无引用也不会被 GC
}

// Go 1.23+，无引用的 Timer/Ticker 会被 GC 回收
// 但最佳实践仍然是显式 Stop
```

> **注意**：这些变更受 `GODEBUG=asynctimerchan` 控制。`asynctimerchan=1` 恢复旧行为。Go 1.23+ 编译的程序默认使用新行为。

### 3.5 标准使用模式

**Timer — 超时控制：**

```go
func doWithTimeout(ctx context.Context, timeout time.Duration) error {
    timer := time.NewTimer(timeout)
    defer timer.Stop()

    select {
    case result := <-workChan:
        return handleResult(result)
    case <-timer.C:
        return fmt.Errorf("operation timed out after %v", timeout)
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

**Ticker — 定期任务：**

```go
func pollStatus(ctx context.Context, interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop() // 必须 Stop！

    for {
        select {
        case <-ticker.C:
            checkStatus()
        case <-ctx.Done():
            return
        }
    }
}
```

**time.AfterFunc — 异步回调：**

```go
// AfterFunc 在独立 goroutine 中执行回调
timer := time.AfterFunc(5*time.Second, func() {
    fmt.Println("5 秒后执行")
})
// 可以取消
timer.Stop()
```

---

## 4 time.After 的陷阱

### 4.1 源码揭示问题

```go
// 源码: time/sleep.go
func After(d Duration) <-chan Time {
    return NewTimer(d).C
}
```

`time.After` 每次调用都创建一个新的 `Timer`，**但返回的是 channel 而不是 Timer 本身**，因此调用者无法 `Stop` 它。

### 4.2 在 select 循环中的内存泄漏

```go
// 危险：每次循环迭代都创建一个新 Timer
for {
    select {
    case msg := <-msgChan:
        handle(msg)
    case <-time.After(5 * time.Second):  // 每次都分配！
        log.Println("timeout")
    }
}
```

如果 `msgChan` 每秒来 1000 条消息，每秒就会创建 1000 个 Timer，在 Go 1.23 之前它们在到期前不会被 GC 回收。

**正确做法：复用 Timer**

```go
timer := time.NewTimer(5 * time.Second)
defer timer.Stop()

for {
    select {
    case msg := <-msgChan:
        handle(msg)
        // Go 1.23+ 可直接 Reset
        timer.Reset(5 * time.Second)
    case <-timer.C:
        log.Println("timeout")
        timer.Reset(5 * time.Second)
    }
}
```

> **Go 1.23+ 注意**：虽然未引用的 Timer 现在可以被 GC 回收，但 `time.After` 在循环中仍会造成不必要的内存分配压力。复用 Timer 始终是更好的选择。

---

## 5 时区处理

### 5.1 Location 的内部结构

```go
// 源码: time/zoneinfo.go
type Location struct {
    name string
    zone []zone        // 该时区的所有历史规则
    tx   []zoneTrans   // 转换时间点（夏令时等）
    extend string      // POSIX TZ 扩展规则
}

type zone struct {
    name   string // "CST", "EST", "PDT" 等
    offset int    // 相对 UTC 的秒数偏移
    isDST  bool   // 是否为夏令时
}
```

**时区数据来源**（按优先级）：

```
1. $ZONEINFO 环境变量指向的 zip 文件
2. runtime.GOROOT()/lib/time/zoneinfo.zip（Go 安装目录）
3. 操作系统时区数据库（/usr/share/zoneinfo 等）
4. time/tzdata 包（嵌入式时区数据，import _ "time/tzdata"）
```

### 5.2 LoadLocation 的性能

```go
loc, err := time.LoadLocation("Asia/Shanghai")
```

`LoadLocation` 内部有缓存机制：

```go
// 源码简化
var loadLocationCache struct {
    sync.Mutex
    m map[string]*Location
}

func LoadLocation(name string) (*Location, error) {
    // 快路径：UTC 和 Local 直接返回
    if name == "" || name == "UTC" {
        return UTC, nil
    }
    if name == "Local" {
        return Local, nil
    }
    // 检查缓存
    if z, ok := cache[name]; ok {
        return z, nil
    }
    // 慢路径：从文件系统加载时区数据
    // ...
}
```

**生产建议**：虽然有缓存，但 `LoadLocation` 首次调用需要磁盘 I/O。在程序启动时加载并缓存常用时区：

```go
var (
    Shanghai *time.Location
    NewYork  *time.Location
)

func init() {
    var err error
    Shanghai, err = time.LoadLocation("Asia/Shanghai")
    if err != nil {
        log.Fatal("load Shanghai timezone:", err)
    }
    NewYork, err = time.LoadLocation("America/New_York")
    if err != nil {
        log.Fatal("load NewYork timezone:", err)
    }
}
```

### 5.3 时区转换

```go
now := time.Now()                           // 本地时间
utc := now.UTC()                            // 转为 UTC
shanghai := now.In(Shanghai)                // 转为上海时间

// 注意：In() 不改变时间点，只改变展示方式
// now、utc、shanghai 代表同一个时刻
fmt.Println(now.Equal(utc))                 // true
fmt.Println(now.Equal(shanghai))            // true
```

### 5.4 time.Date 与时区

```go
// 创建特定时区的时间
t := time.Date(2024, 7, 15, 14, 30, 0, 0, Shanghai)

// 陷阱：夏令时切换期间，某些"本地时间"可能不存在或有歧义
// 例如美国东部时间 2024-03-10 02:30:00 不存在（跳过了）
t = time.Date(2024, 3, 10, 2, 30, 0, 0, NewYork)
fmt.Println(t) // Go 会自动调整到合理的时间
```

---

## 6 格式化与解析

### 6.1 参考时间的设计哲学

Go 使用一个**具体的参考时间**作为格式模板，而非 `YYYY-MM-DD` 这类抽象占位符：

```
Mon Jan 2 15:04:05 MST 2006
```

为什么是这个时间？将其按位置编号：

```
月份:  Jan        → 1 月
日期:  2          → 2 号
小时:  15 (3 PM)  → 15 时
分钟:  04         → 04 分
秒:    05         → 05 秒
年份:  2006       → 2006 年
时区:  MST        → Mountain Standard Time (UTC-7)
```

编号序列：**1 2 3 4 5 6 7**（月日时分秒年时区偏移）。这是一种助记设计。

### 6.2 常用格式

```go
// 预定义常量（源码: time/format.go）
const (
    Layout      = "01/02 03:04:05PM '06 -0700"
    ANSIC       = "Mon Jan _2 15:04:05 2006"
    RFC3339     = "2006-01-02T15:04:05Z07:00"
    RFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"
    DateTime    = "2006-01-02 15:04:05"      // Go 1.20+
    DateOnly    = "2006-01-02"               // Go 1.20+
    TimeOnly    = "15:04:05"                 // Go 1.20+
)

// 使用
t := time.Now()
fmt.Println(t.Format(time.RFC3339))          // "2024-01-15T10:30:00+08:00"
fmt.Println(t.Format(time.DateTime))         // "2024-01-15 10:30:00"
fmt.Println(t.Format("2006年01月02日"))        // "2024年01月15日"
```

### 6.3 Format vs AppendFormat

```go
// Format 返回 string（分配新内存）
s := t.Format(time.RFC3339)

// AppendFormat 追加到已有 []byte（减少分配）
buf := make([]byte, 0, 64)
buf = t.AppendFormat(buf, time.RFC3339)
```

在高频格式化场景（如日志、序列化），`AppendFormat` 可以显著减少内存分配。详见 `performance/format-vs-appendformat/`。

### 6.4 Parse vs ParseInLocation

这是最常见的时区陷阱之一：

```go
// time.Parse：无时区信息时默认 UTC
t1, _ := time.Parse("2006-01-02 15:04:05", "2024-01-15 10:30:00")
fmt.Println(t1.Location()) // UTC ← 注意！

// time.ParseInLocation：无时区信息时使用指定时区
t2, _ := time.ParseInLocation("2006-01-02 15:04:05", "2024-01-15 10:30:00", time.Local)
fmt.Println(t2.Location()) // Local

// t1 和 t2 代表不同的时刻！
fmt.Println(t1.Equal(t2))  // false（除非本地时区就是 UTC）
```

**规则**：当解析不含时区的时间字符串时，**必须使用 `ParseInLocation`**，否则会默认 UTC，导致时间偏差。详见 `trap/parse-without-timezone/`。

---

## 7 time.Sleep 与定时器选型

### 7.1 time.Sleep 的实现

```go
// 源码: time/sleep.go
func Sleep(d Duration) {
    if d <= 0 {
        return
    }
    t := new(timer)
    t.init(goroutineReady, nil)
    t.reset(when, 0)
    gopark(...)  // 让出 goroutine，等待 timer 唤醒
}
```

`time.Sleep` 本质上也是创建一个 runtime timer，然后将当前 goroutine 挂起。与 `Timer.C` 的区别是它不经过 channel。

### 7.2 选型指南

| 场景 | 推荐方案 | 原因 |
|------|---------|------|
| 简单等待 | `time.Sleep` | 最简洁，无需管理资源 |
| select 超时 | `time.NewTimer` | 可在 select 中使用，可 Stop |
| 定期执行 | `time.NewTicker` | 自动重复，无累积漂移 |
| 异步回调 | `time.AfterFunc` | 在独立 goroutine 执行 |
| 请求超时 | `context.WithTimeout` | 与 Go 生态深度集成 |
| 绝对截止 | `context.WithDeadline` | 适合"在某个时间点前完成" |

### 7.3 context.WithTimeout vs Timer

```go
// context.WithTimeout 内部就是用 Timer 实现的
func WithTimeout(parent Context, timeout time.Duration) (Context, CancelFunc) {
    return WithDeadline(parent, time.Now().Add(timeout))
}

func WithDeadline(parent Context, d time.Time) (Context, CancelFunc) {
    // ...
    c := &timerCtx{deadline: d}
    // 启动一个 Timer，到期时自动取消 context
    c.timer = time.AfterFunc(dur, func() {
        c.cancel(true, DeadlineExceeded, nil)
    })
    return c, func() { c.cancel(true, Canceled, nil) }
}
```

**关键**：`context.WithTimeout` 返回的 `cancel` 函数**必须调用**，否则 Timer 资源不会提前释放：

```go
ctx, cancel := context.WithTimeout(parentCtx, 5*time.Second)
defer cancel() // 必须调用！即使操作提前完成

result, err := doWork(ctx)
```

---

## 8 时间比较与运算

### 8.1 比较方法

```go
a := time.Now()
b := a.Add(time.Hour)

// 比较
a.Before(b)    // true
a.After(b)     // false
a.Equal(b)     // false（注意：用 Equal 而非 ==）

// 为什么不用 == ？
// == 会比较所有字段（包括 Location 指针），同一时刻不同时区会返回 false
t1 := time.Date(2024, 1, 1, 8, 0, 0, 0, Shanghai)
t2 := t1.UTC()
fmt.Println(t1 == t2)    // false（Location 不同）
fmt.Println(t1.Equal(t2)) // true（同一时刻）
```

### 8.2 时间运算

```go
now := time.Now()

// 加减 Duration
future := now.Add(24 * time.Hour)
past := now.Add(-30 * time.Minute)

// 加减年月日（处理了月份天数差异）
nextMonth := now.AddDate(0, 1, 0)
nextYear := now.AddDate(1, 0, 0)

// 两个时间的差值
elapsed := future.Sub(now) // Duration

// 从 now 到现在过了多久
time.Since(start) // 等价于 time.Now().Sub(start)

// 到 deadline 还有多久
time.Until(deadline) // 等价于 deadline.Sub(time.Now())
```

### 8.3 AddDate 的陷阱

```go
// 1月31日 + 1个月 = ?
t := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
t = t.AddDate(0, 1, 0)
fmt.Println(t) // 2024-03-02（不是2月29日！）
// 原因：先计算2月31日，溢出后规范化为3月2日
```

---

## 9 生产环境最佳实践

### 9.1 存储与传输

```go
// 存储：始终用 UTC
createdAt := time.Now().UTC()

// 传输：使用 RFC3339 格式
jsonTime := createdAt.Format(time.RFC3339Nano)

// 数据库：使用 TIMESTAMP WITH TIME ZONE 类型
// Go 的 database/sql 会自动处理 time.Time ↔ TIMESTAMPTZ 转换
```

### 9.2 定时任务防漂移

```go
// 错误：Sleep 会累积漂移
for {
    doWork()              // 假设执行 200ms
    time.Sleep(time.Second) // 实际间隔 1.2s，漂移越来越大
}

// 正确：使用 Ticker
ticker := time.NewTicker(time.Second)
defer ticker.Stop()
for range ticker.C {
    doWork()              // Ticker 基于绝对时间触发，不会漂移
}
```

### 9.3 优雅关闭定时器

```go
func worker(ctx context.Context) {
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            log.Println("worker stopped")
            return
        case <-ticker.C:
            doWork()
        }
    }
}
```

### 9.4 测试中的时间控制

生产代码应避免直接调用 `time.Now()`，而是通过接口注入：

```go
// 定义时钟接口
type Clock interface {
    Now() time.Time
    Since(t time.Time) time.Duration
    NewTicker(d time.Duration) *time.Ticker
}

// 生产实现
type RealClock struct{}
func (RealClock) Now() time.Time                          { return time.Now() }
func (RealClock) Since(t time.Time) time.Duration         { return time.Since(t) }
func (RealClock) NewTicker(d time.Duration) *time.Ticker  { return time.NewTicker(d) }

// 测试实现
type FakeClock struct {
    now time.Time
}
func (f *FakeClock) Now() time.Time                          { return f.now }
func (f *FakeClock) Since(t time.Time) time.Duration         { return f.now.Sub(t) }
func (f *FakeClock) NewTicker(d time.Duration) *time.Ticker  { /* 可控的 Ticker */ }
func (f *FakeClock) Advance(d time.Duration)                 { f.now = f.now.Add(d) }
```

### 9.5 时区处理清单

- [ ] 服务器统一使用 UTC 时区（`TZ=UTC`）
- [ ] 数据库存储 TIMESTAMPTZ，不要用 TIMESTAMP
- [ ] API 输入输出使用 RFC3339 格式
- [ ] 使用 `ParseInLocation` 而非 `Parse` 解析时间
- [ ] 比较时间用 `Equal`，不要用 `==`
- [ ] 启动时加载并缓存需要的 `*Location`
- [ ] Docker 镜像中包含时区数据（或 `import _ "time/tzdata"`）

---

## 10 常见陷阱速查

| 陷阱 | 原因 | 详情 |
|------|------|------|
| Ticker 不 Stop 导致泄漏 | Ticker 在 runtime 中注册，不 Stop 不会被清理（Go <1.23） | `trap/ticker-not-stopped/` |
| Timer Reset 前未 drain | channel 缓冲区有旧值，Reset 后立即读取会拿到旧值（Go <1.23） | `trap/timer-reset-drain/` |
| time.After 在循环中泄漏 | 每次创建新 Timer 且无法 Stop | `trap/time-after-in-loop/` |
| 时区比较出错 | `==` 比较 Location 指针，同一时刻不同时区返回 false | `trap/timezone-comparison/` |
| Parse 默认 UTC | 无时区的字符串被解析为 UTC 而非本地时间 | `trap/parse-without-timezone/` |
| Sleep 导致时间漂移 | Sleep 的间隔 = sleep + 执行时间，逐渐累积 | `trap/ticker-drift/` |

## 11 性能基准

| 实验 | 对比内容 | 详情 |
|------|---------|------|
| Timer vs Ticker | 循环创建 Timer vs 复用 Ticker | `performance/timer-vs-ticker/` |
| Format vs AppendFormat | 格式化字符串分配对比 | `performance/format-vs-appendformat/` |
| time.Now 开销 | time.Now() 的调用成本 | `performance/time-now-cost/` |
| Sleep vs Timer | time.Sleep vs NewTimer 的性能对比 | `performance/sleep-vs-timer/` |
