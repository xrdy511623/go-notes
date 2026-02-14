
---
详解 Go 语言中的 Fuzzing
---

Fuzzing（模糊测试）是一种通过自动生成随机输入来发现程序缺陷的测试技术。Go 1.18 将 Fuzzing 纳入标准工具链，极大降低了使用门槛。


# 1 什么是 Fuzzing

传统单元测试依赖开发者预设的测试用例，覆盖已知的正常路径和边界条件。但对于复杂的输入空间，开发者很难穷尽所有"刁钻"组合。Fuzzing 通过**自动生成大量随机输入**来弥补这个盲区。

| 维度 | 单元测试 | Fuzzing |
|------|---------|---------|
| 输入来源 | 开发者预先定义 | 引擎自动生成和变异 |
| 探索性 | 验证已知行为 | 发现未知缺陷 |
| 侧重点 | 功能正确性 | 健壮性、安全性、边界问题 |
| 运行时间 | 毫秒级 | 秒到小时级（持续探索） |

Fuzzing 最擅长发现的问题：
- **panic** — 空指针、越界、类型断言失败等
- **边界逻辑错误** — off-by-one、整数溢出
- **解析器漏洞** — 畸形输入导致的安全问题（如 CVE）
- **不变性违反** — encode/decode 不一致、状态不一致


# 2 Go 1.18+ 内置 Fuzzing 支持

## 2.1 Fuzz 函数签名

```go
func FuzzXxx(f *testing.F) {
    // 1. 添加种子语料
    f.Add(seedArgs...)

    // 2. 定义 Fuzzing 执行体
    f.Fuzz(func(t *testing.T, args...) {
        // 调用被测函数 + 检查不变性
    })
}
```

**命名规则**：必须以 `Fuzz` 开头（如 `FuzzParseAge`），与 `Test`、`Benchmark` 对应。

## 2.2 f.Add 支持的类型

Go fuzzing 引擎只支持以下基本类型作为 `f.Add()` 和 `f.Fuzz(func(...))` 的参数：

```
string, []byte, bool, byte, rune,
int, int8, int16, int32, int64,
uint, uint8, uint16, uint32, uint64,
float32, float64
```

**不支持** struct、slice（除 `[]byte`）、map、指针等复合类型。如果需要 fuzz 复合类型，需要在 `f.Fuzz` 内部用基本类型参数构造。

## 2.3 运行命令

```bash
# 运行指定的 Fuzz 目标，持续 30 秒
go test -fuzz=FuzzParseAge -fuzztime=30s

# 运行所有 Fuzz 目标，持续 1 分钟
go test -fuzz=. -fuzztime=1m

# 限制并行度（默认 GOMAXPROCS）
go test -fuzz=FuzzParseAge -fuzztime=30s -parallel=4

# 仅运行种子语料作为回归测试（不做 fuzzing，CI 中使用）
go test -run=FuzzParseAge

# 重现特定失败
go test -run=FuzzParseAge/testdata_corpus_file_name
```

## 2.4 testing.F 的方法

| 方法 | 用途 |
|------|------|
| `f.Add(args...)` | 添加种子语料 |
| `f.Fuzz(fn)` | 定义 fuzzing 执行体（每个 FuzzXxx 只能调用一次） |
| `f.Skip(args...)` | 跳过当前 fuzz 目标 |
| `f.Helper()` | 标记为辅助函数（错误报告时跳过） |
| `f.Cleanup(fn)` | 注册清理函数 |
| `f.Log/Logf` | 输出日志 |


# 3 基础示例：边界 Bug 发现

最简单的 Fuzzing 用法——发现函数的边界逻辑错误。

```go
// parser.go — 故意在边界上留了 Bug: age > 150 应该是 age > 149
func ParseAge(ageStr string) (int, error) {
    age, err := strconv.Atoi(ageStr)
    if err != nil { return 0, err }
    if age < 0 || age > 150 { // Bug: 150 被错误接受
        return 0, fmt.Errorf("age %d out of range", age)
    }
    return age, nil
}
```

```go
// parser_test.go
func FuzzParseAge(f *testing.F) {
    f.Add("0")
    f.Add("149")
    f.Add("-1")
    f.Add("abc")
    f.Add("")

    f.Fuzz(func(t *testing.T, ageStr string) {
        age, err := ParseAge(ageStr)
        if err != nil { return }

        // 不变性：成功时 age 必须在 [0, 149]
        if age < 0 || age >= 150 {
            t.Errorf("ParseAge(%q) = %d, want 0 <= age < 150", ageStr, age)
        }
    })
}
```

运行结果：

```
$ go test -fuzz=FuzzParseAge -fuzztime=5s
--- FAIL: FuzzParseAge (0.14s)
    --- FAIL: FuzzParseAge (0.00s)
        parser_test.go:38: ParseAge("150") = 150, want 0 <= age < 150
    Failing input written to testdata/fuzz/FuzzParseAge/1b484383a67174f3
```

Fuzzing 引擎自动发现了输入 `"150"` 触发了边界 Bug，并将 crashing input 保存到 `testdata/` 目录。

> 完整代码 → [parser.go](parser.go) + [parser_test.go](parser_test.go)


# 4 Round-trip 模式（黄金模式）

Round-trip 是 Fuzzing 最强大的模式：**encode(x) → bytes → decode(bytes) → y → 断言 x == y**。

不需要知道"正确答案"是什么，只需验证一个数学性质：`decode(encode(x)) == x`。

```go
func FuzzRoundTrip(f *testing.F) {
    f.Add(uint8(1), "hello", int32(100))

    f.Fuzz(func(t *testing.T, typ uint8, name string, score int32) {
        original := Record{Type: typ, Name: name, Score: score}

        data, err := Encode(original)
        if err != nil { t.Fatal(err) }

        decoded, err := Decode(data)
        if err != nil { t.Fatal(err) }

        // Round-trip 不变性
        if decoded != original {
            t.Errorf("round-trip failed: %+v → %+v", original, decoded)
        }
    })
}
```

**适用场景**：
- JSON / XML / Protobuf 序列化
- 自定义二进制协议编解码
- 压缩/解压缩（gzip, zstd, snappy）
- 加密/解密

配合 "Decode 不 panic" 的 fuzz：

```go
func FuzzDecodeNoPanic(f *testing.F) {
    f.Add([]byte{})
    f.Fuzz(func(t *testing.T, data []byte) {
        Decode(data) // 只确保不 panic，不检查返回值
    })
}
```

> 完整代码 → [roundtrip/codec.go](roundtrip/codec.go) + [roundtrip/codec_test.go](roundtrip/codec_test.go)


# 5 []byte 解析器 Fuzz

对于接受 `[]byte` 的解析器，Fuzzing 的核心目标是：**对任意输入，要么返回合法结果，要么返回错误，绝不 panic**。

```go
func FuzzParseMessage(f *testing.F) {
    f.Add([]byte{1, 0x01, 0, 0})          // 合法 PING
    f.Add([]byte{})                        // 空输入
    f.Add([]byte{0xff})                    // 非法 version
    f.Add([]byte{1, 0x02, 0xff, 0xff})    // payload 长度超大

    f.Fuzz(func(t *testing.T, data []byte) {
        msg, err := ParseMessage(data)
        if err != nil { return }

        // 不变性：解析成功时字段必须合法
        if msg.Version != 1 && msg.Version != 2 {
            t.Errorf("invalid version: %d", msg.Version)
        }
        if len(msg.Payload) > 4096 {
            t.Errorf("payload too large: %d", len(msg.Payload))
        }
    })
}
```

真实世界中大量安全漏洞（CVE）都是解析器对畸形输入处理不当导致的。Fuzzing 是发现这类问题最高效的手段。

> 完整代码 → [byteparser/protocol.go](byteparser/protocol.go) + [byteparser/protocol_test.go](byteparser/protocol_test.go)


# 6 多参数 Fuzz

`f.Add()` 支持多个参数，引擎会独立变异每个参数：

```go
func FuzzFormatRecord(f *testing.F) {
    // 三个参数: string, int, bool
    f.Add("Alice", 1, false)
    f.Add("Bob", 999999, true)
    f.Add("", 0, false)

    f.Fuzz(func(t *testing.T, name string, id int, uppercase bool) {
        formatted := FormatRecord(name, id, uppercase)

        // Round-trip: 格式化后应能解析回来
        parsedName, parsedID, err := ParseRecord(formatted)
        if err != nil {
            t.Fatalf("ParseRecord(%q) failed: %v", formatted, err)
        }

        if parsedID < 0 || parsedID > 999999 {
            t.Errorf("parsed id %d out of range", parsedID)
        }
        if parsedName == "" {
            t.Error("parsed name is empty")
        }
    })
}
```

**注意**：`f.Add()` 的参数类型和数量必须与 `f.Fuzz(func(t, ...))` 中的参数完全匹配，否则运行时 panic。

> 完整代码 → [multiparam/format.go](multiparam/format.go) + [multiparam/format_test.go](multiparam/format_test.go)


# 7 Differential Fuzz（差分模糊测试）

用两个不同的实现处理同一输入，比较结果是否一致。如果不一致，说明至少有一个实现有 Bug。

```go
func FuzzSplitDiff(f *testing.F) {
    f.Add("hello,world", ",")
    f.Add("a::b::c", "::")

    f.Fuzz(func(t *testing.T, s, sep string) {
        if sep == "" { t.Skip() }

        got := splitManual(s, sep)    // 手动实现
        want := strings.Split(s, sep) // 标准库

        if len(got) != len(want) {
            t.Fatalf("result length mismatch: %d vs %d", len(got), len(want))
        }
        for i := range got {
            if got[i] != want[i] {
                t.Errorf("part[%d]: %q vs %q", i, got[i], want[i])
            }
        }
    })
}
```

**适用场景**：
- 重构后新旧实现的对比验证
- 标准库 vs 第三方库的一致性检查
- 不同算法实现的等价性验证
- 跨语言移植后的行为对比

> 完整代码 → [differential/diff_test.go](differential/diff_test.go)


# 8 Corpus 管理与 CI 集成

## 8.1 Corpus 目录结构

```
mypackage/
├── mycode.go
├── mycode_test.go
└── testdata/
    └── fuzz/
        └── FuzzMyFunction/
            ├── 1b484383a67174f3    ← Fuzzing 发现的 crashing input
            ├── seed_input_1        ← 手动添加的种子文件
            └── ...
```

Fuzzing 发现的 crashing input 自动写入 `testdata/fuzz/FuzzXxx/` 目录。这些文件**应该提交到 Git**，作为回归测试的一部分。

每次 `go test -run=FuzzXxx`（不带 `-fuzz`）会自动运行 `testdata/` 中的所有 corpus 文件，确保已修复的 Bug 不会复发。

## 8.2 Fuzzing cache

除了 `testdata/`，Fuzzing 引擎还会在 `$GOCACHE/fuzz/` 中缓存大量"有趣"的输入（覆盖新路径但未触发失败的输入）。这些缓存不需要提交到 Git，但可以加速后续 fuzzing。

```bash
# 查看 cache 路径
go env GOCACHE

# 清理 fuzzing cache
go clean -fuzzcache
```

## 8.3 CI 集成策略

Fuzzing 不适合在 CI 中无限运行。推荐策略：

```yaml
# GitHub Actions 示例
jobs:
  test:
    steps:
      # 1. 常规测试 + corpus 回归（每次 PR 都运行）
      - run: go test ./...

  fuzz:
    # 2. 短时间 fuzzing（每次 PR 运行 30 秒）
    steps:
      - run: go test -fuzz=. -fuzztime=30s ./mypackage/

  fuzz-extended:
    # 3. 长时间 fuzzing（每周定时运行）
    if: github.event_name == 'schedule'
    steps:
      - run: go test -fuzz=. -fuzztime=10m ./mypackage/
```

**要点**：
- `go test -run=FuzzXxx`（无 `-fuzz`）用于回归测试，每次 CI 都跑
- `go test -fuzz=. -fuzztime=30s` 用于增量 fuzzing，每次 PR 跑
- 长时间 fuzzing 安排在定时任务中（如每周跑 10 分钟）
- 发现的 crashing input 应提交到 `testdata/` 作为回归测试


# 9 高级参数与调优

## 9.1 常用参数

```bash
# 基本用法
go test -fuzz=FuzzXxx -fuzztime=30s

# 限制并行度（默认 GOMAXPROCS）
go test -fuzz=FuzzXxx -fuzztime=30s -parallel=2

# 限制 minimization 时间（减小 crashing input 的时间上限）
go test -fuzz=FuzzXxx -fuzztime=30s -fuzzminimizetime=10s

# 组合使用
go test -fuzz=. -fuzztime=1m -parallel=4 -v ./...
```

## 9.2 参数说明

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-fuzz=regexp` | 无 | 匹配要运行的 Fuzz 目标 |
| `-fuzztime=duration` | 无限 | Fuzzing 持续时间（如 `30s`, `1m`, `100x`） |
| `-fuzzminimizetime` | 60s | 最小化 crashing input 的时间限制 |
| `-parallel=n` | GOMAXPROCS | Fuzzing 并行度 |
| `-run=regexp` | 所有 | 不带 `-fuzz` 时用于回归测试 |

`-fuzztime` 也支持次数限制：`-fuzztime=1000x` 表示运行 1000 次迭代后停止。

## 9.3 覆盖率引导

Go 的 Fuzzing 引擎是**覆盖率引导（Coverage-guided）**的：
1. 执行一个变异输入
2. 收集代码覆盖率信息
3. 如果发现新的代码路径，保留该输入到 corpus
4. 基于 corpus 中的输入继续变异

这意味着引擎会自动趋向于探索更多代码路径，而不是盲目随机。**高质量的种子语料**可以帮助引擎更快地到达深层代码路径。

## 9.4 种子语料的最佳实践

```go
f.Add(validInput)           // 合法输入（帮助引擎理解正常路径）
f.Add(edgeCaseInput)        // 边界值（0, MaxInt, 空字符串等）
f.Add(knownBadInput)        // 已知的问题输入（历史 Bug）
f.Add(structurallyDifferent) // 结构不同的输入（帮助探索不同分支）
```

**不需要**：提供大量相似的种子（引擎的变异策略会自动生成）。


# 10 从 crash 到修复的完整工作流

```
1. 运行 Fuzzing → 发现 crash
   $ go test -fuzz=FuzzParseAge -fuzztime=30s
   --- FAIL: FuzzParseAge
       Failing input written to testdata/fuzz/FuzzParseAge/abc123

2. 查看 crashing input
   $ cat testdata/fuzz/FuzzParseAge/abc123
   go test fuzz v1
   string("150")

3. 重现 crash（确认问题）
   $ go test -run=FuzzParseAge/abc123
   --- FAIL

4. 修复代码
   age > 150  →  age > 149

5. 确认修复（运行回归测试）
   $ go test -run=FuzzParseAge
   PASS  ← testdata 中的 corpus 全部通过

6. 提交 testdata 中的 crashing input 到 Git
   $ git add testdata/fuzz/
   $ git commit -m "fix: ParseAge 边界 Bug + 回归测试 corpus"

7. 继续 Fuzzing（确认没有更多问题）
   $ go test -fuzz=FuzzParseAge -fuzztime=1m
   PASS
```

**关键**：`testdata/fuzz/` 目录下的 crashing input 必须提交到 Git，作为永久的回归测试。每次 `go test` 都会自动运行这些 corpus，确保已修复的 Bug 不会复发。


# 11 Fuzzing 最佳实践

## 11.1 选择 Fuzz 目标

**优先选择**：
- 解析器（JSON、XML、Protobuf、自定义二进制格式）
- 编解码器（序列化/反序列化）
- 处理用户输入的函数
- 任何接受 `[]byte` 或 `string` 的复杂逻辑

**不适合**：
- 依赖外部状态（数据库、网络）的函数
- 执行非常慢的函数（每次调用 >100ms）
- 输入空间极小的简单函数（如算术运算）

## 11.2 编写有效的断言

```go
f.Fuzz(func(t *testing.T, input []byte) {
    // 1. 最基本：不 panic（Fuzzing 自动检测）

    // 2. 错误处理：返回错误时提前返回
    result, err := MyFunc(input)
    if err != nil { return }

    // 3. 不变性检查：成功时验证结果的性质
    if result < 0 {
        t.Errorf("negative result: %d", result)
    }

    // 4. Round-trip：encode → decode → 比较
    // 5. Differential：两个实现比较
})
```

## 11.3 Fuzzing 的局限性

- **不能替代单元测试**：Fuzzing 是补充，不是替代。业务逻辑的正确性仍需单元测试验证
- **不确定性**：同一个 Bug 可能需要运行数秒或数小时才能被发现
- **类型限制**：Go 内置 fuzzing 只支持基本类型参数
- **状态无关**：Go 内置 fuzzing 不支持有状态的测试序列（如"先创建用户，再删除用户"）

对于需要有状态 fuzzing 或更复杂的类型支持，可以考虑第三方工具如 [go-fuzz](https://github.com/dvyukov/go-fuzz) 或 [Atheris](https://github.com/google/atheris)（Python）。

## 11.4 go-fuzz vs Go 1.18+ 内置 Fuzzing

| 维度 | go-fuzz | Go 1.18+ 内置 |
|------|---------|---------------|
| 安装 | 需要单独安装 | 内置，零配置 |
| API | 自定义 `Fuzz(data []byte) int` | 标准 `testing.F` |
| 参数类型 | 仅 `[]byte` | 多种基本类型 |
| 覆盖率引导 | 是 | 是 |
| corpus 管理 | 手动 | 自动（`testdata/fuzz/`） |
| CI 集成 | 复杂 | 简单（`go test`） |

对于新项目，推荐使用 Go 1.18+ 内置 Fuzzing。go-fuzz 在需要更细粒度控制或旧版本 Go 时仍有价值。
