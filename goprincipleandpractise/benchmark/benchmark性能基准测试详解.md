
---
benchmark性能基准测试详解
---

# 1 benchmark性能基准测试规则

测试文件名必须以_test.go结尾；
测试函数名必须以BenchmarkXxx开始，Xxx通常是待测试的函数名
在命令行下使用go test -bench=.即可开始性能测试
b.N表示循环执行的次数，而N值不用程序员特别关心，N值是动态调整的，直到可靠地计算出程序执行时间后才会停止，具体
执行次数会在执行结束后打印出来。

**注意：benchmark 函数体内必须按 `b.N` 或 `b.Loop()` 循环执行待测逻辑。**
如果只调用一次待测函数，结果会失真，不能反映真实吞吐。

```go
// 错误写法：只执行一次
func BenchmarkFib(b *testing.B) { Fib(50) }

// 正确写法（Go 1.24+）
func BenchmarkFib(b *testing.B) {
    for b.Loop() {
        Fib(50)
    }
}
```

go test 命令默认是不运行 benchmark 用例的，如果我们想运行 benchmark 用例，则需要加上 -bench 参数。例如：

```shell
$ go test -bench=. .
goos: darwin
goarch: amd64
pkg: example
BenchmarkFib-8               200           5865240 ns/op
PASS
ok      example 1.782s
```

-bench 参数支持传入一个正则表达式，匹配到的用例才会得到执行，例如，只运行以 Fib 结尾的 benchmark 用例：

```shell
$ go test -bench='Fib$' .
goos: darwin
goarch: amd64
pkg: example
BenchmarkFib-8               202           5980669 ns/op
PASS
ok      example 1.813s
```

# 2 benchmark 是如何工作的?
benchmark 用例的参数 b *testing.B，有个属性 b.N 表示这个用例需要运行的次数。b.N 对于每个用例都是不一样的。
那这个值是如何决定的呢？b.N 从 1 开始，如果该用例能够在 1s 内完成，b.N 的值便会增加，再次执行。b.N 的值大概以 1, 2, 
3, 5, 10, 20, 30, 50, 100 这样的序列递增，越到后面，增加得越快。我们仔细观察上述例子的输出：

```shell
BenchmarkFib-8                                 1        48182215750 ns/op              8 B/op          1 allocs/op
```

BenchmarkFib-8 中的 -8 即 GOMAXPROCS，默认等于 CPU 核数。可以通过 -cpu 参数改变 GOMAXPROCS，-cpu 支持传入一个
列表作为参数，例如：

```shell
$ go test -bench='Fib$' -cpu=2,4 .
goos: darwin
goarch: amd64
pkg: example
BenchmarkFib-2               206           5774888 ns/op
BenchmarkFib-4               205           5799426 ns/op
PASS
ok      example 3.563s
```
在这个例子中，改变 CPU 的核数对结果几乎没有影响，因为这个 Fib 的调用是串行的。

```shell
BenchmarkAppendIndexed-8        1000000000               0.6790 ns/op          8 B/op          0 allocs/op
```

我们对上面的输出结果做一个简要说明:
第一列 1000000000 表示该测试用例执行了1000000000次，即b.N的值；
第二列 0.6790 ns/op 表示每次执行测试用例花费0.6790纳秒；
第三列 8 B/op  表示每次执行测试用例需要分配8b的内存；
第四列 0 allocs/op  表示每次执行测试用例需要分配0次内存；

## 2.1 提升准确度
对于性能测试来说，提升测试准确度的一个重要手段就是增加测试的次数。我们可以使用 -benchtime 和 -count 两个参数达到这个目的。
benchmark 的默认时间是 1s，那么我们可以使用 -benchtime 指定为 5s。例如：

```shell
go test -bench='Fib$' -benchtime=5s .
goos: darwin
goarch: amd64
pkg: example
BenchmarkFib-8              1033           5769818 ns/op
PASS
ok      example 6.554s
```

注意：实际执行的时间是 6.5s，比 benchtime 的 5s 要长，这是因为测试用例的编译、执行、销毁等是需要时间的。
将 -benchtime 设置为 5s，用例执行次数也变成了原来的 5 倍，每次函数调用时间仍为 0.6s，几乎没有变化。

-benchtime 的值除了是时间外，还可以是具体的次数。例如，执行 30 次可以用 -benchtime=30x

```shell
$ go test -bench='Fib$' -benchtime=50x .
goos: darwin
goarch: amd64
pkg: example
BenchmarkFib-8                50           6121066 ns/op
PASS
ok      example 0.319s
```

-count 参数可以用来设置 benchmark 的轮数。例如，进行 3 轮 benchmark。

```shell
$ go test -bench='Fib$' -benchtime=5s -count=3 .
goos: darwin
goarch: amd64
pkg: example
BenchmarkFib-8               975           5946624 ns/op
BenchmarkFib-8              1023           5820582 ns/op
BenchmarkFib-8               961           6096816 ns/op
PASS
ok      example 19.463s
```

## 2.2 内存分配情况
-benchmem 参数可以度量内存分配的次数。内存分配次数与性能也是息息相关的，例如不合理的切片容量，将导致内存重新分配，
带来不必要的开销。

我们可以使用 -benchmem 参数看到内存分配的情况：

```shell
go test -bench=^Bench -benchmem .
goos: darwin
goarch: arm64
pkg: go-notes/slice/performance
BenchmarkAppend-8                     1        2590832125 ns/op        35762614168 B/op       1901334 allocs/op
BenchmarkAppendAllocated-8      1000000000               0.6862 ns/op          8 B/op          0 allocs/op
BenchmarkAppendIndexed-8        1000000000               0.6542 ns/op          8 B/op          0 allocs/op
```

## 2.3 测试不同的输入
不同的函数复杂度不同，O(1)，O(n)，O(n^2) 等，利用 benchmark 验证复杂度一个简单的方式，是构造不同的输入。
例如：

```golang

// generate_test.go
package main

import (
	"math/rand"
	"testing"
	"time"
)

func generate(n int) []int {
	rand.Seed(time.Now().UnixNano())
	nums := make([]int, 0)
	for i := 0; i < n; i++ {
		nums = append(nums, rand.Int())
	}
	return nums
}
func benchmarkGenerate(i int, b *testing.B) {
	for n := 0; n < b.N; n++ {
		generate(i)
	}
}

func BenchmarkGenerate1000(b *testing.B)    { benchmarkGenerate(1000, b) }
func BenchmarkGenerate10000(b *testing.B)   { benchmarkGenerate(10000, b) }
func BenchmarkGenerate100000(b *testing.B)  { benchmarkGenerate(100000, b) }
func BenchmarkGenerate1000000(b *testing.B) { benchmarkGenerate(1000000, b) }
```

这里，我们实现一个辅助函数 benchmarkGenerate 允许传入参数 i，并构造了 4 个不同输入的 benchmark 用例。运行结果如下：

```shell
$ go test -bench=. .                                                       
goos: darwin
goarch: amd64
pkg: example
BenchmarkGenerate1000-8            34048             34643 ns/op
BenchmarkGenerate10000-8            4070            295642 ns/op
BenchmarkGenerate100000-8            403           3230415 ns/op
BenchmarkGenerate1000000-8            39          32083701 ns/op
PASS
ok      example 6.597s
```
通过测试结果可以发现，输入变为原来的 10 倍，函数每次调用的时长也差不多是原来的 10 倍，这说明复杂度是线性的。


# 2.4 比较两次运行结果
benchstat 是 Go 官方性能工具，用来比较两次基准测试（benchmark）结果的统计差异，可自动计算：

平均值（mean）
中位数
标准差
ns/op、B/op、allocs/op 的变化比例

非常适合以下场景：
✔ 优化代码后想验证是否真的变快
✔ 对比不同算法版本
✔ 调整 GC、参数后验证效果
✔ 分析锁竞争、内存分配变化

**安装**

```shell
go install golang.org/x/perf/cmd/benchstat@latest
```

**使用**

运行第一次基准测试（baseline）

```shell
go test -bench=. -benchmem -count=10 > old.txt
```

修改代码后运行第二次基准测试（new version）

```shell
go test -bench=. -benchmem -count=10 > new.txt
```

用 benchstat 对比两个结果

```shell
benchstat old.txt new.txt
```

输出结果如下：

```shell
 benchstat old.txt new.txt
goos: darwin
goarch: arm64
pkg: go-notes/goprincipleandpractise/slice/performance
cpu: Apple M4
          │        old.txt        │                new.txt                │
          │        sec/op         │    sec/op     vs base                 │
Append-10   2110301604.0000n ± 2%   0.5722n ± 6%  -100.00% (p=0.000 n=10)

          │       old.txt        │               new.txt               │
          │         B/op         │    B/op     vs base                 │
Append-10   35762721200.000 ± 0%   8.000 ± 0%  -100.00% (p=0.000 n=10)

          │   old.txt   │               new.txt                │
          │  allocs/op  │  allocs/op   vs base                 │
Append-10   1.902M ± 0%   0.000M ± 0%  -100.00% (p=0.000 n=10)
```


# 3 benchmark 注意事项

## 3.1 ResetTimer
如果在 benchmark 开始前，需要一些准备工作，如果准备工作比较耗时，则需要将这部分代码的耗时忽略掉。比如下面的例子：

```golang
func BenchmarkFib(b *testing.B) {
	time.Sleep(time.Second * 3) // 模拟耗时准备任务
	for n := 0; n < b.N; n++ {
		fib(30) // run fib(30) b.N times
	}
}
```

运行结果是:
```shell
$ go test -bench='Fib$' -benchtime=50x .
goos: darwin
goarch: amd64
pkg: example
BenchmarkFib-8                50          65912552 ns/op
PASS
ok      example 6.319s
```
50次调用，每次调用约 0.66s，是之前的 0.06s 的 11 倍。究其原因，受到了耗时准备任务的干扰。我们需要用 ResetTimer 
屏蔽掉：

```golang
func BenchmarkFib(b *testing.B) {
	time.Sleep(time.Second * 3) // 模拟耗时准备任务
	b.ResetTimer() // 重置定时器
	for n := 0; n < b.N; n++ {
		fib(30) // run fib(30) b.N times
	}
}
```

此时，运行结果便恢复正常了，每次调用约 0.06s。

```shell
$ go test -bench='Fib$' -benchtime=50x .
goos: darwin
goarch: amd64
pkg: example
BenchmarkFib-8                50           6187485 ns/op
PASS
ok      example 6.330s
```

## 3.2 StopTimer & StartTimer
还有一种情况，每次函数调用前后需要一些准备工作和清理工作，我们可以使用 StopTimer 暂停计时以及使用 StartTimer 开始 计时。

例如，如果测试一个冒泡函数的性能，每次调用冒泡函数前，需要随机生成一个数字序列，这是非常耗时的操作，这种场景下，就需要使用
StopTimer 和 StartTimer 避免将这部分时间计算在内。

例如：

```golang

// sort_test.go
package main

import (
	"math/rand"
	"testing"
	"time"
)

func generateWithCap(n int) []int {
	rand.Seed(time.Now().UnixNano())
	nums := make([]int, 0, n)
	for i := 0; i < n; i++ {
		nums = append(nums, rand.Int())
	}
	return nums
}

func bubbleSort(nums []int) {
	for i := 0; i < len(nums); i++ {
		for j := 1; j < len(nums)-i; j++ {
			if nums[j] < nums[j-1] {
				nums[j], nums[j-1] = nums[j-1], nums[j]
			}
		}
	}
}

func BenchmarkBubbleSort(b *testing.B) {
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		nums := generateWithCap(10000)
		b.StartTimer()
		bubbleSort(nums)
	}
}
```

执行该用例，每次排序耗时约 0.1s。

```shell
$ go test -bench='Sort$' .
goos: darwin
goarch: amd64
pkg: example
BenchmarkBubbleSort-8                  9         113280509 ns/op
PASS
ok      example 1.146s
```

## 3.3 b.ReportAllocs

除了使用命令行的 `-benchmem` 参数外，也可以在代码中调用 `b.ReportAllocs()`，效果等同，好处是不依赖运行时传参，
适合团队统一规范：

```golang
func BenchmarkMarshal(b *testing.B) {
	b.ReportAllocs()
	data := &User{Name: "Alice", Age: 30}
	for i := 0; i < b.N; i++ {
		json.Marshal(data)
	}
}
```

# 4 子基准测试 (b.Run)

使用 `b.Run` 可以在一个 Benchmark 函数中创建多个子基准测试，类似于单元测试中的 `t.Run`。
相比第 2.3 节中为每个输入规模手写独立函数的方式，`b.Run` 更加灵活简洁：

```golang
func BenchmarkGenerate(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				generate(size)
			}
		})
	}
}
```

运行结果会自动带上子测试名称：
```shell
BenchmarkGenerate/size=1000-8          34048         34643 ns/op
BenchmarkGenerate/size=10000-8          4070        295642 ns/op
BenchmarkGenerate/size=100000-8          403       3230415 ns/op
BenchmarkGenerate/size=1000000-8          39      32083701 ns/op
```

可以用正则只运行特定的子测试：
```shell
go test -bench='BenchmarkGenerate/size=10000' .
```

**优势**
- 代码更简洁，不需要为每个输入规模写独立的函数。
- 输出结果层次清晰，自动以 `/` 分隔父子关系。
- 便于通过正则过滤运行特定的子测试。


# 5 并行基准测试 (b.RunParallel)

`b.RunParallel` 用于测试并发场景下的性能，它会启动多个 goroutine 并行执行基准测试。
适用于测试锁竞争、并发安全的数据结构、连接池等场景。

```golang
func BenchmarkConcurrentMap(b *testing.B) {
	m := sync.Map{}
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			m.Store(i, i)
			i++
		}
	})
}
```

与普通 benchmark 的对比：
```golang
// 串行测试：衡量单线程性能
func BenchmarkMutexSerial(b *testing.B) {
	var mu sync.Mutex
	var count int
	for i := 0; i < b.N; i++ {
		mu.Lock()
		count++
		mu.Unlock()
	}
}

// 并行测试：衡量多核竞争下的性能
func BenchmarkMutexParallel(b *testing.B) {
	var mu sync.Mutex
	var count int
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			count++
			mu.Unlock()
		}
	})
}
```

可以结合 `-cpu` 参数观察不同并发度下的性能变化：
```shell
go test -bench='Parallel' -cpu=1,2,4,8 .
```


# 6 从 benchmark 生成性能剖析文件

benchmark 可以直接生成 pprof 剖析文件，便于定位热点函数和内存瓶颈。

**生成 CPU 剖析**
```shell
go test -bench=. -cpuprofile=cpu.out
go tool pprof cpu.out
```

**生成内存剖析**
```shell
go test -bench=. -memprofile=mem.out
go tool pprof mem.out
```

**在 pprof 交互模式中常用命令**
```shell
(pprof) top10          # 查看耗时/内存占用前10的函数
(pprof) list funcName  # 查看指定函数的逐行分析
(pprof) web            # 生成调用图（需要安装graphviz）
```

这是将 benchmark 和 pprof 结合使用的标准工作流：先通过 benchmark 发现性能问题，再通过 pprof 定位到具体的代码行。


# 7 小结

| 功能 | 命令/API | 使用场景 |
|------|----------|----------|
| 基础运行 | `go test -bench=.` | 运行所有基准测试 |
| 内存统计 | `-benchmem` 或 `b.ReportAllocs()` | 查看内存分配情况 |
| 提升准确度 | `-benchtime=5s -count=10` | 获取统计可靠的结果 |
| 结果对比 | `benchstat old.txt new.txt` | 验证优化效果 |
| 排除初始化 | `b.ResetTimer()` | 跳过耗时的准备工作 |
| 每轮排除 | `b.StopTimer()` / `b.StartTimer()` | 每次迭代排除非测试逻辑 |
| 子基准测试 | `b.Run(name, func)` | 组织多组输入的测试 |
| 并行测试 | `b.RunParallel(func)` | 测试并发竞争场景 |
| 性能剖析 | `-cpuprofile` / `-memprofile` | 定位热点函数 |

**最佳实践**
- 始终使用 `-benchmem`，内存分配次数往往比 CPU 耗时更能反映真实性能瓶颈。
- 使用 `-count=10` 并配合 `benchstat` 分析，避免因噪声得出错误结论。
- 先 benchmark 发现问题，再 pprof 定位根因，最后 benchstat 验证优化效果。
