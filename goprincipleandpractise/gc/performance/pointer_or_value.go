package performance

type Person struct {
	id   int
	age  int
	name string
}

type Item struct {
	id  int
	val [40960]int
}

func newPersonValueSlice(n int) []Person {
	s := make([]Person, n)
	for i := 0; i < n; i++ {
		s[i] = Person{}
	}
	return s
}

func newPersonPointerSlice(n int) []*Person {
	s := make([]*Person, n)
	for i := 0; i < n; i++ {
		s[i] = &Person{}
	}
	return s
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
