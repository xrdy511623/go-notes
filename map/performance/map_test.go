package performance

import "testing"

/*
不断向map添加元素的操作会触发map的扩容；
提前分配好空间可以减少内存拷贝和rehash的消耗；
结论: 根据实际需求提前分配好存储空间有利于提高性能
*/

func BenchmarkWithoutPreAlloc(b *testing.B) { WithoutPreAlloc(10000) }
func BenchmarkWithPreAlloc(b *testing.B)    { PreAlloc(10000) }
