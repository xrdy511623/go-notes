package performance

import (
	"fmt"
)

/*
测试结果参考images目录下的slice-performance.png

通过benchMark性能对比测试，可以发现从上到下，性能是越来越好，每次操作需要分配的内存空间以及内存分配次数都有了大幅下降。
原因在于通过append的方式向切片添加数据，由于不确定切片后续的长度和容量能到多少，所以一直需要不断的扩容和分配内存，反之，如果
一开始就用make函数初始化slice,明确切片的容量，预先分配内存，那么就可以避免后续添加数据时频繁的扩容和分配内存，大幅提高
性能，而第三个按索引下标赋值的方式，相比于append添加数据显然性能更优，那是因为append会有额外的开销。

所以我们得出结论:
对切片预先分配内存可以提升性能;
直接使用index下表索引赋值而非append添加数据可以提升性能。

BenchmarkAppendLoop-10          1000000000               0.6290 ns/op          6 B/op          0 allocs/op
BenchmarkAppendSpread-10        1000000000               0.1634 ns/op          1 B/op          0 allocs/op
如果不需要过滤切片，直接将一个切片里的元素全部拷贝到另一个切片里，可以使用 ... 展开，性能更优。
*/

func Append(num int) {
	for i := 0; i < num; i++ {
		s := []int{}
		for j := 0; j < 10000; j++ {
			s = append(s, j)
		}
	}
}

func AppendAllocated(num int) {
	for i := 0; i < num; i++ {
		s := make([]int, 0, 10000)
		for j := 0; j < 10000; j++ {
			s = append(s, j)
		}
	}
}

func AppendIndexed(num int) {
	for i := 0; i < num; i++ {
		s := make([]int, 10000)
		for j := 0; j < 10000; j++ {
			s[j] = j
		}
	}
}

func AppendLoop(num int) {
	autoFields := make([]string, num)
	for i := range autoFields {
		autoFields[i] = "field"
	}

	for i := 0; i < num; i++ {
		setParts := []string{}
		for _, f := range autoFields {
			setParts = append(setParts, f)
		}
		// 防止编译器优化
		_ = setParts
	}
}

func AppendSpread(num int) {
	autoFields := make([]string, num)
	for i := range autoFields {
		autoFields[i] = "field"
	}

	for i := 0; i < num; i++ {
		setParts := []string{}
		// 使用 ... 展开
		setParts = append(setParts, autoFields...)
		// 防止编译器优化
		_ = setParts
	}
}

func GenerateSlice(n int) []int {
	s := make([]int, n)
	for i := 0; i < n; i++ {
		s[i] = i
	}
	return s
}

/*
测试结果参考images目录下的slice-bce.png

Bce这种写法明显性能更优(测试案例中性能提升30%)，因为在给i累加值之前，它预先检查了s[n-1]的索引下标是否越界，
这样，之后每次累加时，就不需要再检查切片的索引下标是否越界了。

结论: 如果能确定访问到的slice长度，可以先执行一次让编译器去做优化，省去后续每次做索引下标是否越界检查的开销。
*/

func Normal(n int) {
	v := 0
	s := GenerateSlice(n)
	for i := range s {
		v += s[i]
	}
	fmt.Println(v)
}

func Bce(n int) {
	v := 0
	s := GenerateSlice(n)
	_ = s[n-1]
	for i := range s {
		v += s[i]
	}
	fmt.Println(v)
}
