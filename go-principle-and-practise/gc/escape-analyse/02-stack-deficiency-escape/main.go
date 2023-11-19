package main

type Item struct {
	id  int
	val [40960]int
}

func newItemValueSlice(n int) []Item {
	s := make([]Item, n)
	for i := 0; i < n; i++ {
		s[i] = Item{
			i,
			[40960]int{},
		}
	}
	return s
}

func newItemPointerSlice(n int) []*Item {
	s := make([]*Item, n)
	for i := 0; i < n; i++ {
		s[i] = &Item{
			i,
			[40960]int{},
		}
	}
	return s
}

func MakeSlice() {
	s := make([]int, 10000, 10000)
	for i := range s {
		s[i] = i
	}
}

func main() {
	MakeSlice()
	newItemValueSlice(10000)
	newItemPointerSlice(10000)
}
