package performance

import "testing"

/*
对比接口装箱（boxing）的内存分配开销。

执行命令:

	go test -run '^$' -bench '^BenchmarkAlloc' -benchtime=3s -count=5 -benchmem .

关注指标:
  - allocs/op: 是否产生堆分配
  - B/op: 每次操作分配的字节数

预期结论:
 1. 小值（<=指针大小）装箱到interface{}时，Go运行时有优化，可能零分配。
 2. 大结构体装箱必然产生堆分配，分配大小等于结构体大小。
 3. 指针接收者通过接口调用不会额外拷贝值，值接收者会产生拷贝。
 4. 已经是指针的值赋给接口不需要额外分配。
*/

var sinkVal int

// baseline: 直接操作结构体，无接口参与
func BenchmarkAllocNoBoxing(b *testing.B) {
	s := SmallValue{X: 42}
	for b.Loop() {
		sinkVal = s.Value()
	}
}

// 小值装箱到 interface{}
func BenchmarkAllocBoxingSmall(b *testing.B) {
	for b.Loop() {
		var iface interface{} = SmallValue{X: 42}
		sinkVal = iface.(SmallValue).X
	}
}

// 大结构体装箱到 interface{}
func BenchmarkAllocBoxingLarge(b *testing.B) {
	for b.Loop() {
		var iface interface{} = LargeValue{A: 1, B: 2, C: 3, D: 4, E: 5, F: 6, G: 7, H: 8}
		sinkVal = iface.(LargeValue).A
	}
}

// 值接收者通过接口调用（SmallValue实现Valuer）
func BenchmarkAllocValueReceiverSmall(b *testing.B) {
	s := SmallValue{X: 42}
	var iface Valuer = s
	for b.Loop() {
		sinkVal = iface.Value()
	}
}

// 指针接收者通过接口调用（*SmallValue实现PtrValuer）
func BenchmarkAllocPointerReceiverSmall(b *testing.B) {
	s := &SmallValue{X: 42}
	var iface PtrValuer = s
	for b.Loop() {
		sinkVal = iface.PtrValue()
	}
}

// 值接收者通过接口调用（LargeValue——大结构体拷贝开销）
func BenchmarkAllocValueReceiverLarge(b *testing.B) {
	l := LargeValue{A: 1, B: 2, C: 3, D: 4, E: 5, F: 6, G: 7, H: 8}
	var iface Valuer = l
	for b.Loop() {
		sinkVal = iface.Value()
	}
}

// 指针接收者通过接口调用（*LargeValue——无拷贝）
func BenchmarkAllocPointerReceiverLarge(b *testing.B) {
	l := &LargeValue{A: 1, B: 2, C: 3, D: 4, E: 5, F: 6, G: 7, H: 8}
	var iface PtrValuer = l
	for b.Loop() {
		sinkVal = iface.PtrValue()
	}
}
