
---
fuzzing模糊测试
---

传统的单元测试依赖于开发者预先设计好的测试用例，这些用例通常覆盖了已知的正常路径、错误路径和一些常见的边界条件。
但是，对于复杂的输入空间，开发者很难穷尽所有可能的“刁钻”组合，那些未被预料到的输入往往是隐藏 Bug 和安全漏洞的温床。
Fuzzing（模糊测试）正是一种旨在解决这个问题的自动化测试技术。


# 1  什么是 Fuzzing（模糊测试）？
Fuzzing 是一种软件测试技术，它通过向被测试程序（通常是某个函数或 API）提供大量自动生成的、通常是无效的、非预期的或随机的数据
作为输入，然后监控程序的行为，观察是否会发生崩溃（panic）、断言失败、错误、未定义行为（如无限循环、内存损坏）或安全漏洞（如缓冲区溢出）。

Fuzzing 测试与普通测试在下面几方面存在显著差异：
输入生成：普通测试的输入由开发者预先定义；Fuzzing 的输入由 Fuzzing 引擎自动生成和变异。
探索性：普通测试验证已知行为；Fuzzing 旨在探索未知行为和发现隐藏缺陷。
目标：普通测试通常关注功能正确性；Fuzzing 更侧重于健壮性、安全性和发现边缘案例。

可以看到，Fuzzing 测试是常规测试的重要补充，Go 团队在语言演进过程中十分重视 Fuzzing 测试技术，早在 2015 年，
Go goroutine 调度器的设计者、现 Google 工程师的Dmitry Vyukov，就实现了 Go 语言的首个 fuzzing 工具go-fuzz。
Go 1.18 版本开始，Go 更是内置了对 Fuzzing 的支持。接下来，我们就来看看如何在 Go 1.18+ 中实现对代码的 Fuzzing test。


# 2 Go 1.18+ 内置 Fuzzing 支持
从 Go 1.18 版本开始，Go 语言在标准工具链中内置了对 Fuzzing 的原生支持，这使得在 Go 项目中应用 Fuzzing 变得非常方便。
Fuzz 测试函数的签名：一个 Fuzz 测试目标（Fuzz Target）是一个名为 FuzzXxx 的函数，它接受一个 *testing.F 类型的参数。

```go
go func FuzzMyFunction(f *testing.F) {     // ... Fuzzing logic ... }
```

f.Add(args…) 添加种子语料（Seed Corpus）：种子语料是一些由开发者提供的、合法的、有代表性的输入示例。Fuzzing 引擎
会以这些种子为基础，通过各种变异策略（如位翻转、删除、插入、拼接等）来生成新的测试输入。提供高质量的种子语料可以显著提高
Fuzzing 发现 Bug 的效率。

Fuzzing 执行体：这是 Fuzz 测试的核心。fuzzFn 是一个回调函数，它会被 Fuzzing 引擎用不同的输入
（包括初始的种子语料和引擎生成的变异输入）反复调用。

第一个参数 t *testing.T：与单元测试中的 t 类似，用于报告失败（如 t.Fatal、t.Error）。如果 fuzzFn 中发生 panic，
或者调用了 t.Fatal，Fuzzing 引擎会认为发现了一个“有趣”的输入（crashing input），并将其保存下来。

后续参数：这些参数的类型由开发者定义，Fuzzing 引擎会尝试为这些类型的参数生成和变异数据。f.Add() 提供的种子语料的类型
必须与这些参数匹配。


```go
f.Fuzz(fuzzFn func(t *testing.T, originalCorpusArgs..., generatedArgs...)) 
```

go test -fuzz=FuzzTargetName 命令的使用：使用此命令来启动对特定 Fuzz 目标（如 FuzzMyFunction）的 Fuzzing 过程。
Fuzzing 会持续运行，直到被手动停止（Ctrl+C）、发生崩溃、或者达到某个预设的时间 / 迭代限制。 当 Fuzzing 发现导致问题
的输入时，它会将这个输入保存到一个名为 testdata/fuzz/FuzzTargetName/ 的文件中，并报告失败。开发者随后可以用这个
crashing input 来编写一个普通的单元测试，以便稳定复现和调试该 Bug。


让我们来看一个非常基础的例子，假设我们有一个简单的函数，它尝试解析一个表示年龄的字符串，并期望年龄在合理范围内。

```go
package simpleparser

import (
    "fmt"
    "strconv"
)

// ParseAge parses a string into an age (integer).
// It expects the age to be between 0 and 150.
func ParseAge(ageStr string) (int, error) {
    if ageStr == "" {
        return 0, fmt.Errorf("age string cannot be empty")
    }
    age, err := strconv.Atoi(ageStr)
    if err != nil {
        return 0, fmt.Errorf("not a valid integer: %w", err)
    }
    if age < 0 || age > 150 { // Let's introduce a potential bug for "> 150" for fuzzing to find
        // if age < 0 || age >= 150 { // Corrected logic for ">="
        return 0, fmt.Errorf("age %d out of reasonable range (0-149)", age)
    }
    return age, nil
}
```

注意：上面的代码中，我们故意在年龄上限的判断上留了一个小问题（age > 150 而不是 age >= 150 或者 age > 149），
看看 Fuzzing 能否帮助我们发现相关的边界问题。

下面是针对 ParseAge 的 Fuzz 测试代码：

```go
package simpleparser

import (
    "testing"
    "unicode/utf8" // We might check for valid strings if ParseAge expects it
)

func FuzzParseAge(f *testing.F) {
    // Add seed corpus: valid ages, edge cases, invalid inputs
    f.Add("0")
    f.Add("1")
    f.Add("149") // Edge case for upper bound (based on "> 150" bug, this is valid)
    f.Add("-1")
    f.Add("abc")      // Not an integer
    f.Add("")         // Empty string
    f.Add("1000")     // Out of range
    f.Add(" 77 ")     // String with spaces (Atoi handles this)
    f.Add("\x80test") // Invalid UTF-8 prefix - strconv.Atoi might handle or error early

    // The Fuzzing execution function
    f.Fuzz(func(t *testing.T, ageStr string) {
        // Call the function being fuzzed
        age, err := ParseAge(ageStr)

        // Define our expectations / invariants
        if err != nil {
            t.Logf("ParseAge(%q) returned error: %v (this might be expected for fuzzed inputs)", ageStr, err)
            return
        }

        if age < -1000 || age > 1000 { // Arbitrary broad check for successfully parsed ages
            t.Errorf("ParseAge(%q) resulted in an unexpected age %d without error", ageStr, age)
        }

        if utf8.ValidString(ageStr) {
            if age < 0 || age >= 150 {
                t.Errorf("Successfully parsed age %d for input %q is out of the *absolute* expected range 0-150", age, ageStr)
            }
        }
    })
}
```


通过下面命令运行 Fuzz 测试：

```shell
 go test -fuzz=FuzzParseAge -fuzztime=5s   
fuzz: elapsed: 0s, gathering baseline coverage: 0/142 completed
fuzz: elapsed: 0s, gathering baseline coverage: 142/142 completed, now fuzzing with 10 workers
fuzz: minimizing 31-byte failing input file
fuzz: elapsed: 0s, minimizing
--- FAIL: FuzzParseAge (0.14s)
    --- FAIL: FuzzParseAge (0.00s)
        parser_test.go:38: Successfully parsed age 150 for input "150" is out of the *absolute* expected range 0-150
    
    Failing input written to testdata/fuzz/FuzzParseAge/1b484383a67174f3
    To re-run:
    go test -run=FuzzParseAge/1b484383a67174f3
FAIL
exit status 1
FAIL    go-notes/goprincipleandpractise/fuzzingtest     0.360s
```


**Fuzzing 如何帮助发现问题**


Panic 发现：如果某个生成的 ageStr（例如，一个超长的数字字符串导致 strconv.Atoi 内部问题，尽管不太可能）导致 ParseAge 或 Atoi 
发生 panic，Fuzzing 会捕获它。

边界条件与逻辑错误：ParseAge 的逻辑是 if age < 0 || age > 150 { return err }。这意味着输入"150"
会被 ParseAge 认为是合法的（150 > 150 为 false）。在 f.Fuzz 的逻辑中，如果我们写一个断言，例如，
我们期望 ParseAge 能成功解析 0 到 150（包含 150）之间的所有数字字符串。如果 ParseAge 内部的检查是
age > 150 才会报错，那么当 Fuzzing 引擎用种子"150"调用时，ParseAge 会成功返回 150。
但如果 ParseAge 的检查是 age >= 150 就会报错，那么 Fuzzing 用种子"150"调用时，err 会非 nil，
我们的 f.Fuzz 逻辑会提前返回。 关键在于 f.Fuzz 内部的断言如何定义“正确性”。如果 Fuzzing 引擎生成了一个字符串
（或我们提供了种子）如 “150”，ParseAge 由于 Bug age > 150（应该 age >= 150 如果 150 是上限的话，或者 age > 149 如果 149 是上限）
可能会错误地接受或拒绝它。

修正 ParseAge 的 Bug：假设我们希望年龄范围是 0 <= age <= 149。那么 ParseAge 应为 if age < 0 || age > 149 { err }。
此时，种子 f.Add(“149”) 应该成功。
种子 f.Add(“150”) 应该导致 ParseAge 返回错误。
如果 Fuzzing 引擎生成的输入（或种子）"150"被 ParseAge 错误地解析成功了（例如，如果 ParseAge 的检查是 age > 150），
而我们的 f.Fuzz 中的断言是 if age < 0 || age > 149 { t.Error(…) }，那么当 age 是 150 时，这个断言就会触发，
Fuzzing 就会报告这个不一致。


```shell
# Fuzzing发现问题后
ls testdata/fuzz/FuzzParseAge/

# 重现特定失败
go test -run=FuzzParseAge/1b484383a67174f3
--- FAIL: FuzzParseAge (0.00s)
    --- FAIL: FuzzParseAge/1b484383a67174f3 (0.00s)
        parser_test.go:38: Successfully parsed age 150 for input "150" is out of the *absolute* expected range 0-150
FAIL
exit status 1
FAIL    go-notes/goprincipleandpractise/fuzzingtest     0.277s
```


这个简单的例子展示了 Fuzzing 如何通过自动生成输入并检查不变性（或预期行为）来帮助发现代码中的逻辑缺陷和边界问题。
那么如何编写更为有效的 Fuzz 测试，能帮助开发人员更快找到一些边界条件相关的问题呢？我们接下来继续看。


# 3 编写有效的 Fuzz 测试

## 3.1 选择合适的 Fuzz 目标
首先是选择合适的 Fuzz 目标函数。Fuzzing 最适合用于测试那些处理外部的、不可信的或复杂格式输入的函数。例如：
解析函数（如解析 JSON、XML、Protobuf、自定义二进制格式等）。
编解码函数。
处理用户输入的 API 端点逻辑（如果能将核心处理逻辑提取出来）。
任何接受字节切片、字符串或复杂结构体作为输入的且内部逻辑复杂的函数。
而依赖大量外部状态（如数据库）、执行非常慢、或者输入空间过于简单（如简单的算术运算）的函数，可能不是 Fuzzing 的最佳目标。


## 3.2 提供有意义的种子语料（f.Add）
其次是提供有意义的种子语料（f.Add）。种子语料应覆盖一些已知的、合法的，以及可能触发边界条件的输入。例如，对于一个解析器，
种子可以包括一些简单的有效输入、空的输入、以及一些结构上略有不同的有效输入。Fuzzing 引擎会基于这些种子进行变异。


## 3.3 定义 Fuzzing 期望
接着是在 Fuzzing 执行体（f.Fuzz）中编写清晰的检查逻辑。
核心任务：在 fuzzFn 中，你需要调用被测试的函数，并对其行为进行检查。
检查内容：
不应 Panic：这是 Fuzzing 最容易发现的问题。如果被测函数因某个输入而 panic，Fuzzing 引擎会自动捕获并报告。
错误处理：如果函数对于无效输入应该返回错误，验证是否返回了错误，并且错误类型或内容是否符合预期。
不变性检查（Invariants）：检查某些在函数执行前后应该保持不变的属性。例如，如果一个函数对数据进行编码后再解码，结果应该与原始数据相同。
行为一致性：例如，用两种不同的方法处理同一个输入，结果应该一致。
避免副作用：如果被测函数不能有副作用（如修改全局状态），可以检查这些状态。

使用 t *testing.T 的方法（如 t.Errorf、t.Fatalf）来报告任何不符合预期的行为。


## 3.4 理解 Fuzzing 引擎的工作方式
最后是理解 Fuzzing 引擎的工作方式。Go 的内置 Fuzzing 引擎是覆盖率引导（Coverage-guided）的。这意味着它会优先选择那些能够
触发新代码路径的输入进行变异和保留，从而更有效地探索代码。

Fuzzing 是现代软件测试工具箱中一个越来越重要的组成部分，Go 将其集成到标准工具链中，极大地降低了开发者使用这一强大技术的门槛。