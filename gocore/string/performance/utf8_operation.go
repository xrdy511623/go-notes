package performance

import (
	"unicode/utf8"
)

var (
	utf8Sink    int
	utf8StrSink string
)

// CountByteLen 使用 len() 获取字节长度
func CountByteLen(s string) int {
	return len(s)
}

// CountRuneLen 使用 utf8.RuneCountInString 获取字符（rune）数量
func CountRuneLen(s string) int {
	return utf8.RuneCountInString(s)
}

// CountRuneByRange 使用 for range 遍历来计数 rune
func CountRuneByRange(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

// IterateByByte 按字节遍历字符串
func IterateByByte(s string) int {
	sum := 0
	for i := 0; i < len(s); i++ {
		sum += int(s[i])
	}
	return sum
}

// IterateByRune 按 rune 遍历字符串（for range）
func IterateByRune(s string) int {
	sum := 0
	for _, r := range s {
		sum += int(r)
	}
	return sum
}

// IterateByDecodeRune 使用 utf8.DecodeRuneInString 手动解码
func IterateByDecodeRune(s string) int {
	sum := 0
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		sum += int(r)
		i += size
	}
	return sum
}

// ReverseByRune 按 rune 反转字符串（正确方式）
func ReverseByRune(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// ReverseByByte 按字节反转字符串（对 ASCII 安全，对 UTF-8 多字节字符错误）
func ReverseByByte(s string) string {
	b := []byte(s)
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}
