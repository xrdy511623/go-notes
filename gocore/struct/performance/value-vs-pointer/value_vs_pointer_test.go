package valuevspointer

import (
	"testing"
	"unsafe"
)

/*
值传递 vs 指针传递对不同大小 struct 的性能影响

执行命令:
go test -bench=^Bench -benchtime=3s -benchmem .
goos: darwin
goarch: arm64
pkg: go-notes/goprincipleandpractise/struct/performance/value-vs-pointer
cpu: Apple M4
BenchmarkSmallValue-10          1000000000               3.000 ns/op           0 B/op          0 allocs/op
BenchmarkSmallPointer-10        1000000000               3.000 ns/op           0 B/op          0 allocs/op
BenchmarkMediumValue-10         1000000000               3.000 ns/op           0 B/op          0 allocs/op
BenchmarkMediumPointer-10       1000000000               3.000 ns/op           0 B/op          0 allocs/op
BenchmarkLargeValue-10          253027470               14.22 ns/op            0 B/op          0 allocs/op
BenchmarkLargePointer-10        1000000000               3.000 ns/op           0 B/op          0 allocs/op
PASS
ok      go-notes/goprincipleandpractise/struct/performance/value-vs-pointer     18.869s

对比维度:
  - Small (16B): 值传递 vs 指针传递
  - Medium (128B): 值传递 vs 指针传递
  - Large (1024B): 值传递 vs 指针传递

结论:
  - Small struct (≤64B): 值传递通常更快（栈分配友好，无间接引用开销）
  - Medium struct (64-256B): 差异不大，视具体场景而定
  - Large struct (>256B): 指针传递明显更快（避免大块内存拷贝）
  - 指针传递可能导致逃逸到堆，带来 GC 压力（需 -gcflags="-m" 确认）

选型建议:
  - 小于等于 2-3 个机器字的 struct → 值传递
  - 需要修改 receiver 状态 → 必须用指针
  - 大 struct 或作为接口使用 → 指针传递
  - 不确定时，先用值传递，profile 后再优化
*/

func TestStructSizes(t *testing.T) {
	t.Logf("Small  size=%d", unsafe.Sizeof(Small{}))
	t.Logf("Medium size=%d", unsafe.Sizeof(Medium{}))
	t.Logf("Large  size=%d", unsafe.Sizeof(Large{}))
}

// ---------- Small ----------

func BenchmarkSmallValue(b *testing.B) {
	s := Small{X: 1, Y: 2}
	for b.Loop() {
		sinkInt64 = ProcessSmallValue(s)
	}
}

func BenchmarkSmallPointer(b *testing.B) {
	s := &Small{X: 1, Y: 2}
	for b.Loop() {
		sinkInt64 = ProcessSmallPointer(s)
	}
}

// ---------- Medium ----------

func BenchmarkMediumValue(b *testing.B) {
	s := Medium{}
	s.Data[0] = 1
	s.Data[15] = 2
	for b.Loop() {
		sinkInt64 = ProcessMediumValue(s)
	}
}

func BenchmarkMediumPointer(b *testing.B) {
	s := &Medium{}
	s.Data[0] = 1
	s.Data[15] = 2
	for b.Loop() {
		sinkInt64 = ProcessMediumPointer(s)
	}
}

// ---------- Large ----------

func BenchmarkLargeValue(b *testing.B) {
	s := Large{}
	s.Data[0] = 1
	s.Data[127] = 2
	for b.Loop() {
		sinkInt64 = ProcessLargeValue(s)
	}
}

func BenchmarkLargePointer(b *testing.B) {
	s := &Large{}
	s.Data[0] = 1
	s.Data[127] = 2
	for b.Loop() {
		sinkInt64 = ProcessLargePointer(s)
	}
}
