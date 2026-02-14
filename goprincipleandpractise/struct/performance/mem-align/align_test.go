package memalign

import (
	"testing"
	"unsafe"
)

/*
struct 字段排列顺序对性能的影响

执行命令:

	go test -run '^$' -bench '^Benchmark' -benchtime=3s -count=3 -benchmem .

对比维度:
  - order (8 字节): 字段按对齐倍数从小到大排列，无浪费
  - disOrder (12 字节): 字段交错排列，多 4 字节 padding

结论:
  - disOrder 比 order 多占 50% 内存 (12 vs 8 字节)
  - order 比 disorder 提升CPU性能 23%
  - 大量分配时，disOrder 的 allocs/op 更高
  - 内存占用差异导致缓存效率下降，遍历性能降低
  - 建议：字段按对齐倍数从小到大排列，或使用 fieldalignment 工具自动优化
*/

func TestStructSize(t *testing.T) {
	t.Logf("order    size=%d  align=%d", unsafe.Sizeof(order{}), unsafe.Alignof(order{}))
	t.Logf("disOrder size=%d  align=%d", unsafe.Sizeof(disOrder{}), unsafe.Alignof(disOrder{}))
}

func BenchmarkUseOrderStruct(b *testing.B) {
	for b.Loop() {
		UseOrderStruct(10000)
	}
}

func BenchmarkUseDisOrderStruct(b *testing.B) {
	for b.Loop() {
		UseDisOrderStruct(10000)
	}
}
