package performance

import (
	"fmt"
	"strconv"
	"testing"
)

var convertSink []string

func ConvertIntToStringSprint(s []int) []string {
	n := len(s)
	strSlice := make([]string, n)
	for i := 0; i < n; i++ {
		strSlice[i] = fmt.Sprint(s[i])
	}
	return strSlice
}

func ConvertIntToStringStrconv(s []int) []string {
	n := len(s)
	strSlice := make([]string, n)
	for i := 0; i < n; i++ {
		strSlice[i] = strconv.Itoa(s[i])
	}
	return strSlice
}

func GenerateSlice(n int) []int {
	s := make([]int, n)
	for i := 0; i < n; i++ {
		s[i] = i
	}
	return s
}

func BenchmarkConvert(b *testing.B, f func([]int) []string) {
	s := GenerateSlice(10000)
	b.ResetTimer()
	for b.Loop() {
		convertSink = f(s)
	}
}
