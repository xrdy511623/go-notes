package performance

import "testing"

/*
不断向map添加元素的操作会触发map的扩容；
提前分配好空间可以减少内存拷贝和rehash的消耗；
结论: 根据实际需求提前分配好存储空间有利于提高性能

执行命令:

go test -run '^$' -bench '^Benchmark' -benchtime=3s -count=5 -benchmem .

Apple M4(Go 1.24)下5次均值:

	BenchmarkWithoutPreAlloc-10    13662    266075 ns/op    591484 B/op    79 allocs/op
	BenchmarkWithPreAlloc-10       45901     77906 ns/op    295553 B/op    33 allocs/op

预分配快 3.4 倍，内存分配次数减少 58%。
*/

func BenchmarkWithoutPreAlloc(b *testing.B) {
	for b.Loop() {
		WithoutPreAlloc(10000)
	}
}

func BenchmarkWithPreAlloc(b *testing.B) {
	for b.Loop() {
		PreAlloc(10000)
	}
}
