package performance

import (
	"testing"
)

/*
编译器优化场景验证

执行命令:

	go test -run '^$' -bench 'Compiler|MapLookup|Compare|ConcatConst' -benchmem .

验证逃逸分析:

	go test -gcflags="-m" -run '^$' -bench 'MapLookup' . 2>&1 | grep "does not escape"

对比维度:
  1. map 查找: m[string(b)] 编译器优化 vs 普通 string key 查找
  2. 字符串比较: string(b) == "literal" 编译器优化 vs string(b) == variable
  3. 常量拼接: 编译期完成 vs 运行时拼接

结论:
  - m[string(b)] 被编译器特殊处理，不产生临时 string 分配（0 allocs/op）
  - string(b) == "literal" 同样被优化，不产生内存分配
  - 常量字符串拼接在编译期完成，运行时零开销
  - 这些优化是 Go 编译器 (gc) 特有的，gccgo 可能不支持
*/

var testMap = map[string]int{
	"hello": 1,
	"world": 2,
	"foo":   3,
	"bar":   4,
}

var testKey = []byte("hello")
var testKeyStr = "hello"

func BenchmarkMapLookupByteSlice(b *testing.B) {
	for b.Loop() {
		_, compilerBoolSink = MapLookupByteSlice(testMap, testKey)
	}
}

func BenchmarkMapLookupString(b *testing.B) {
	for b.Loop() {
		_, compilerBoolSink = MapLookupString(testMap, testKeyStr)
	}
}

func BenchmarkCompareByteSliceToLiteral(b *testing.B) {
	key := []byte("hello")
	for b.Loop() {
		compilerBoolSink = CompareByteSliceToLiteral(key)
	}
}

func BenchmarkCompareByteSliceToVar(b *testing.B) {
	key := []byte("hello")
	s := "hello"
	for b.Loop() {
		compilerBoolSink = CompareByteSliceToVar(key, s)
	}
}

func BenchmarkConcatConstant(b *testing.B) {
	for b.Loop() {
		compilerStrSink = ConcatConstant()
	}
}

func BenchmarkConcatVariable(b *testing.B) {
	a := "hello"
	c := ", world"
	for b.Loop() {
		compilerStrSink = ConcatVariable(a, c)
	}
}
