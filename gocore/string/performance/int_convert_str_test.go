package performance

import "testing"

/*
从 CPU 资源消耗来看，fmt 的方式，单次函数调用要 343682ns，而 strconv 的方式，只要 112317ns，节约了 66.7% 左右的 CPU 资源。
从内存消耗来看，fmt 的方式，单次函数调用要 320585 字节内存，而 strconv 的方式，每次函数调用只要 202721 字节内存，节约了
37.5% 左右的内存资源。

go test -benchmem . -bench="Convert$"
16 32 64 128 256 512 896 1408 2048 3072 4096 5376 6912 9472 12288 16384 21760 28672 40960 57344 73728 98304 131072 goos: darwin
goarch: arm64
pkg: go-notes/goprincipleandpractise/string/performance
cpu: Apple M4
BenchmarkSprintConvert-10           3524            343682 ns/op          320585 B/op      19735 allocs/op
BenchmarkStrconvConvert-10         10000            112317 ns/op          202721 B/op       9901 allocs/op
PASS
ok      go-notes/goprincipleandpractise/string/performance      2.571s
*/

func BenchmarkSprintConvert(b *testing.B)  { BenchmarkConvert(b, ConvertIntToStringSprint) }
func BenchmarkStrconvConvert(b *testing.B) { BenchmarkConvert(b, ConvertIntToStringStrconv) }
