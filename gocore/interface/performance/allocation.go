package performance

// SmallValue 小值类型（<=指针大小，运行时可能不分配）
type SmallValue struct {
	X int
}

func (s SmallValue) Value() int     { return s.X }
func (s *SmallValue) PtrValue() int { return s.X }

// LargeValue 大值类型（>指针大小，装箱时一定会堆分配）
type LargeValue struct {
	A, B, C, D, E, F, G, H int
}

func (l LargeValue) Value() int     { return l.A }
func (l *LargeValue) PtrValue() int { return l.A }

// Valuer 用于测试值接收者 vs 指针接收者
type Valuer interface {
	Value() int
}

// PtrValuer 用于测试指针接收者
type PtrValuer interface {
	PtrValue() int
}
