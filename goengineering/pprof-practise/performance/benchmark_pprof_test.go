package performance

import (
	"fmt"
	"strings"
	"testing"
)

// 本文件演示 benchmark + pprof 联动用法
//
// 生成 CPU profile:
//   go test -bench=BenchmarkConcat -cpuprofile=cpu.prof -benchmem ./goprincipleandpractise/pprof-practise/performance/
//   go tool pprof cpu.prof
//
// 生成内存 profile:
//   go test -bench=BenchmarkConcat -memprofile=mem.prof -benchmem ./goprincipleandpractise/pprof-practise/performance/
//   go tool pprof -alloc_objects mem.prof
//
// 生成 trace:
//   go test -bench=BenchmarkConcat -trace=trace.out ./goprincipleandpractise/pprof-practise/performance/
//   go tool trace trace.out

// concatWithPlus 使用 + 拼接字符串（每次创建新字符串，大量内存分配）
func concatWithPlus(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		s += fmt.Sprintf("item%d,", i)
	}
	return s
}

// concatWithBuilder 使用 strings.Builder（预分配，最小化分配次数）
func concatWithBuilder(n int) string {
	var b strings.Builder
	b.Grow(n * 10) // 预估大小，减少扩容
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "item%d,", i)
	}
	return b.String()
}

func BenchmarkConcatPlus(b *testing.B) {
	for i := 0; i < b.N; i++ {
		concatWithPlus(100)
	}
}

func BenchmarkConcatBuilder(b *testing.B) {
	for i := 0; i < b.N; i++ {
		concatWithBuilder(100)
	}
}

/*
运行并生成 profile:
  go test -bench=BenchmarkConcat -cpuprofile=cpu.prof -memprofile=mem.prof -benchmem ./goprincipleandpractise/pprof-practise/performance/

分析 CPU:
  go tool pprof cpu.prof
  (pprof) top
  (pprof) list concatWithPlus   # 可以看到 + 拼接的逐行 CPU 开销

分析内存:
  go tool pprof -alloc_objects mem.prof
  (pprof) top
  (pprof) list concatWithPlus   # 可以看到大量 alloc_objects

对比:
  go tool pprof -http=:8089 -diff_base cpu_before.prof cpu_after.prof
*/
