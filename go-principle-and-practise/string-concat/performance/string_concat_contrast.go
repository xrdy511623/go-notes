package performance

import (
	"bytes"
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

// 为了避免编译器优化，首先实现一个生成长度为 n 的随机字符串的函数。
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// 然后利用这个函数生成字符串str，然后将str拼接N次。在Go语言中，常见的字符串拼接方式有如下5种：

func PlusConcat(n int, str string) string {
	s := ""
	for i := 0; i < n; i++ {
		s += str
	}
	return s
}

func SprintfConcat(n int, str string) string {
	s := ""
	for i := 0; i < n; i++ {
		s = fmt.Sprintf("%s%s", s, str)
	}
	return s
}

func BuilderConcat(n int, str string) string {
	var builder strings.Builder
	for i := 0; i < n; i++ {
		builder.WriteString(str)
	}
	return builder.String()
}

func BufferConcat(n int, s string) string {
	buf := new(bytes.Buffer)
	for i := 0; i < n; i++ {
		buf.WriteString(s)
	}
	return buf.String()
}

func ByteConcat(n int, str string) string {
	buf := make([]byte, 0)
	for i := 0; i < n; i++ {
		buf = append(buf, str...)
	}
	return string(buf)
}

// PreByteConcat 如果长度是可预知的，那么创建 []byte 时，我们还可以预分配切片的容量(cap)。
func PreByteConcat(n int, str string) string {
	buf := make([]byte, 0, n*len(str))
	for i := 0; i < n; i++ {
		buf = append(buf, str...)
	}
	return string(buf)
}

// PreBuilderConcat string.Builder也提供了预分配内存的方式 Grow
func PreBuilderConcat(n int, str string) string {
	var builder strings.Builder
	builder.Grow(n * len(str))
	for i := 0; i < n; i++ {
		builder.WriteString(str)
	}
	return builder.String()
}

func Benchmark(b *testing.B, f func(int, string) string) {
	var str = RandomString(10)
	for i := 0; i < b.N; i++ {
		f(10000, str)
	}
}
