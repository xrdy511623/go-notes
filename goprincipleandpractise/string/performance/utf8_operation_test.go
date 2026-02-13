package performance

import (
	"strings"
	"testing"
)

/*
UTF-8 操作性能对比

执行命令:

	go test -run '^$' -bench 'UTF8|Iterate|Reverse' -benchmem .

对比维度:
  1. 字符计数: len() vs utf8.RuneCountInString() vs for range 计数
  2. 字符串遍历: 按字节 vs 按 rune(for range) vs 按 rune(DecodeRune)
  3. 字符串反转: 按字节 vs 按 rune

结论:
  - len() 是 O(1) 操作，直接读取 stringStruct.len 字段
  - RuneCountInString() 需要遍历整个字符串解码 UTF-8，O(n)
  - 按字节遍历比按 rune 遍历快，因为无需 UTF-8 解码
  - 对于纯 ASCII 字符串，按字节和按 rune 遍历性能接近
  - 反转操作中，[]rune 转换涉及内存分配，比 []byte 慢
*/

// 混合 ASCII + 中文的测试字符串
var mixedString = strings.Repeat("Hello,世界！Go语言 ", 100)

// 纯 ASCII 测试字符串
var asciiString = strings.Repeat("Hello, World! Go Language ", 100)

func BenchmarkUTF8CountByteLen(b *testing.B) {
	for b.Loop() {
		utf8Sink = CountByteLen(mixedString)
	}
}

func BenchmarkUTF8CountRuneLen(b *testing.B) {
	for b.Loop() {
		utf8Sink = CountRuneLen(mixedString)
	}
}

func BenchmarkUTF8CountRuneByRange(b *testing.B) {
	for b.Loop() {
		utf8Sink = CountRuneByRange(mixedString)
	}
}

func BenchmarkIterateByByte(b *testing.B) {
	for b.Loop() {
		utf8Sink = IterateByByte(mixedString)
	}
}

func BenchmarkIterateByRune(b *testing.B) {
	for b.Loop() {
		utf8Sink = IterateByRune(mixedString)
	}
}

func BenchmarkIterateByDecodeRune(b *testing.B) {
	for b.Loop() {
		utf8Sink = IterateByDecodeRune(mixedString)
	}
}

func BenchmarkIterateByByteASCII(b *testing.B) {
	for b.Loop() {
		utf8Sink = IterateByByte(asciiString)
	}
}

func BenchmarkIterateByRuneASCII(b *testing.B) {
	for b.Loop() {
		utf8Sink = IterateByRune(asciiString)
	}
}

func BenchmarkReverseByRune(b *testing.B) {
	for b.Loop() {
		utf8StrSink = ReverseByRune(mixedString)
	}
}

func BenchmarkReverseByByte(b *testing.B) {
	for b.Loop() {
		utf8StrSink = ReverseByByte(mixedString)
	}
}
