package performance

import "testing"

/*
从 CPU 资源消耗来看，fmt 的方式，单次函数调用要 768717ns，而 strconv 的方式，只要 259444ns，节约了 66.7% 左右的 CPU 资源。
从内存消耗来看，fmt 的方式，单次函数调用要 320668 字节内存，而 strconv 的方式，每次函数调用只要 202739 字节内存，节约了
37.5% 左右的内存资源。

go test -benchmem . -bench="Convert$"
16 32 64 128 256 512 1024 1280 1792 2304 3072 4096 5376 6784 9472 12288 16384 20480 27264 40960 57344 73728 98304 122880 goos: darwin
goarch: amd64
pkg: go-notes/go-principle-and-practise/string/performance
cpu: Intel(R) Core(TM) i9-9880H CPU @ 2.30GHz
BenchmarkSprintConvert-16           1311            768717 ns/op          320668 B/op      19735 allocs/op
BenchmarkStrconvConvert-16          4616            259444 ns/op          202739 B/op       9901 allocs/op
PASS
ok      go-notes/go-principle-and-practise/string/performance   2.967s
*/

func BenchmarkSprintConvert(b *testing.B)  { BenchmarkConvert(b, ConvertIntToStringSprint) }
func BenchmarkStrconvConvert(b *testing.B) { BenchmarkConvert(b, ConvertIntToStringStrconv) }
