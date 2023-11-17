package mem_align

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

func UseOrderStruct(n int) {
	s := make([]order, n)
	for i := 0; i < n; i++ {
		s = append(s, order{})
	}
}

func UseDisOrderStruct(n int) {
	s := make([]disOrder, n)
	for i := 0; i < n; i++ {
		s = append(s, disOrder{})
	}
}
