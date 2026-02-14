package valuevspointer

// 值传递 vs 指针传递对不同大小 struct 的性能影响
//
// Go 的 struct 是值类型，赋值和传参都会完整拷贝。
// 对于小 struct，值传递可能比指针传递更快（栈分配，无 GC 压力）。
// 对于大 struct，值传递的拷贝开销会成为瓶颈。

// Small 小 struct (16 字节)
type Small struct {
	X, Y int64
}

// Medium 中等 struct (128 字节)
type Medium struct {
	Data [16]int64
}

// Large 大 struct (1024 字节)
type Large struct {
	Data [128]int64
}

var sinkInt64 int64

func ProcessSmallValue(s Small) int64    { return s.X + s.Y }
func ProcessSmallPointer(s *Small) int64 { return s.X + s.Y }

func ProcessMediumValue(s Medium) int64    { return s.Data[0] + s.Data[15] }
func ProcessMediumPointer(s *Medium) int64 { return s.Data[0] + s.Data[15] }

func ProcessLargeValue(s Large) int64    { return s.Data[0] + s.Data[127] }
func ProcessLargePointer(s *Large) int64 { return s.Data[0] + s.Data[127] }
