package performance

import (
	"strings"
)

var cloneSink string

// SubstringDirect 直接切片获取子字符串（共享底层数组）
func SubstringDirect(s string, n int) string {
	return s[:n]
}

// SubstringClone 使用 strings.Clone 获取独立副本
func SubstringClone(s string, n int) string {
	return strings.Clone(s[:n])
}

// SubstringConcatCopy 使用 string() + []byte 拷贝方式获取独立副本（Go 1.20 之前常用）
func SubstringConcatCopy(s string, n int) string {
	return string([]byte(s[:n]))
}

// GenerateLargeString 生成指定大小的测试字符串
func GenerateLargeString(size int) string {
	return strings.Repeat("a", size)
}
