package performance

/*
测试结果参考 images 目录下的 slice-performance.png。

通过 benchmark 对比可以看到，从 Append -> AppendAllocated -> AppendIndexed，
单次操作分配次数和内存占用通常会下降。
原因是预分配可以减少扩容次数，而在长度已知时直接下标赋值可以减少 append 的额外开销。

前提：
1. 对切片预先分配内存通常可以提升性能；
2. 最终长度已知且无需过滤时，index 赋值通常优于 append。

BenchmarkAppendLoop-10          1000000000               0.6290 ns/op          6 B/op          0 allocs/op
BenchmarkAppendSpread-10        1000000000               0.1634 ns/op          1 B/op          0 allocs/op
如果不需要过滤切片，直接将一个切片里的元素全部拷贝到另一个切片里，可以使用 ... 展开，通常更高效。
*/

func Append(num int) int {
	totalLen := 0
	for i := 0; i < num; i++ {
		s := []int{}
		for j := 0; j < 10000; j++ {
			s = append(s, j)
		}
		totalLen += len(s)
	}
	return totalLen
}

func AppendAllocated(num int) int {
	totalLen := 0
	for i := 0; i < num; i++ {
		s := make([]int, 0, 10000)
		for j := 0; j < 10000; j++ {
			s = append(s, j)
		}
		totalLen += len(s)
	}
	return totalLen
}

func AppendIndexed(num int) int {
	totalLen := 0
	for i := 0; i < num; i++ {
		s := make([]int, 10000)
		for j := 0; j < 10000; j++ {
			s[j] = j
		}
		totalLen += len(s)
	}
	return totalLen
}

func AppendLoop(num int) int {
	totalLen := 0
	autoFields := make([]string, num)
	for i := range autoFields {
		autoFields[i] = "field"
	}

	for i := 0; i < num; i++ {
		setParts := []string{}
		for _, f := range autoFields {
			setParts = append(setParts, f)
		}
		totalLen += len(setParts)
	}
	return totalLen
}

func AppendSpread(num int) int {
	totalLen := 0
	autoFields := make([]string, num)
	for i := range autoFields {
		autoFields[i] = "field"
	}

	for i := 0; i < num; i++ {
		setParts := []string{}
		// 使用 ... 展开
		setParts = append(setParts, autoFields...)
		totalLen += len(setParts)
	}
	return totalLen
}

func GenerateSlice(n int) []int {
	s := make([]int, n)
	for i := 0; i < n; i++ {
		s[i] = i
	}
	return s
}

/*
测试结果参考 images 目录下的 slice_bce.png。

BCE 这类写法在特定场景下可能更优：在循环前先做一次边界检查，编译器可能消除循环内重复的 bounds check。
是否生效取决于代码形态和编译器优化结果，应以实际 benchmark 为准。

结论：如果能确定访问边界，可尝试 BCE 写法减少边界检查开销。
*/

func SumNormal(s []int) int {
	v := 0
	for i := range s {
		v += s[i]
	}
	return v
}

func SumBce(s []int) int {
	if len(s) == 0 {
		return 0
	}
	v := 0
	_ = s[len(s)-1]
	for i := range s {
		v += s[i]
	}
	return v
}
