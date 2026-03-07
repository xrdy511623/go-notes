package performance

import "testing"

/*
go test -bench=Slice$ -benchmem .
goos: darwin
goarch: arm64
pkg: go-notes/gc/performance
BenchmarkNewPersonValueSlice-8          1000000000               0.0000275 ns/op               0 B/op          0 allocs/op
BenchmarkNewPersonPointerSlice-8        1000000000               0.0002953 ns/op               0 B/op          0 allocs/op
BenchmarkNewItemValueSlice-8                   1        2007900792 ns/op        3276881920 B/op        1 allocs/op
BenchmarkNewItemPointerSlice-8          1000000000               0.6230 ns/op          3 B/op          0 allocs/op
PASS
ok      go-notes/gc/performance 14.224s
*/

func BenchmarkNewPersonValueSlice(b *testing.B) {
	newPersonValueSlice(10000)
}

func BenchmarkNewPersonPointerSlice(b *testing.B) {
	newPersonPointerSlice(10000)
}

func BenchmarkNewItemValueSlice(b *testing.B) {
	newItemValueSlice(10000)
}

func BenchmarkNewItemPointerSlice(b *testing.B) {
	newItemPointerSlice(10000)
}
