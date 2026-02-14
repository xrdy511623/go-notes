package memalign

// 内存对齐对 struct 切片操作性能的影响
//
// order 字段按对齐倍数从小到大排列，占 8 字节
// disOrder 字段交错排列，占 12 字节（多出 4 字节 padding）
// 对大量 struct 操作时，内存占用差异 (8 vs 12) 会影响缓存效率

var sinkOrder order
var sinkDisOrder disOrder

type order struct {
	a int8
	b int16
	c int32
}

type disOrder struct {
	a int8
	c int32
	b int16
}

// UseOrderStruct 分配并遍历对齐良好的 struct 切片
func UseOrderStruct(n int) {
	s := make([]order, n)
	for i := range s {
		s[i] = order{a: int8(i), b: int16(i), c: int32(i)}
	}
	sinkOrder = s[n-1]
}

// UseDisOrderStruct 分配并遍历对齐不佳的 struct 切片
func UseDisOrderStruct(n int) {
	s := make([]disOrder, n)
	for i := range s {
		s[i] = disOrder{a: int8(i), c: int32(i), b: int16(i)}
	}
	sinkDisOrder = s[n-1]
}
