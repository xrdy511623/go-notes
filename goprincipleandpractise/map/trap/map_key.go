package main

import (
	"fmt"
	"math"
)

type S struct {
	ID int
}

func main() {
	m := make(map[float64]int)
	m[1.4] = 1
	m[2.4] = 2
	m[math.NaN()] = 3
	m[math.NaN()] = 3

	/*
		例子中定义了一个 key 类型是 float 型的 map，并向其中插入了 4 个 key：1.4， 2.4， NAN，NAN。
		打印的时候也打印出了 4 个 key，如果你知道 NAN != NAN，也就不奇怪了。因为他们比较的结果不相等，自然，在 map 看来就是两个不同的 key 了。
		接着，我们查询了几个 key，发现 NAN 不存在，2.400000000001 也不存在，而 2.4000000000000000000000001 却存在。
		有点诡异，不是吗？
		接着，我通过汇编发现了如下的事实：
		当用 float64 作为 key 的时候，先要将其转成 uint64 类型，再插入 key 中。

		具体是通过 `Float64frombits` 函数完成：
		可以看到，`2.4` 和 `2.4000000000000000000000001` 经过 `math.Float64bits()` 函数转换后的结果是一样的。自然，二者在 map 看来，就是同一个 key 了。

		所以我们的结论是：float 型可以作为 key，但是由于精度的问题，会导致一些诡异的问题，慎用之。
	*/

	for k, v := range m {
		fmt.Printf("[%v, %d] ", k, v)
	}

	fmt.Printf("\nk: %v, v: %d\n", math.NaN(), m[math.NaN()])
	fmt.Printf("k: %v, v: %d\n", 2.400000000001, m[2.400000000001])
	fmt.Printf("k: %v, v: %d\n", 2.4000000000000000000000001, m[2.4000000000000000000000001])

	fmt.Println(math.NaN() == math.NaN())

	fmt.Println(math.Float64bits(2.4))
	fmt.Println(math.Float64bits(2.400000000001))
	fmt.Println(math.Float64bits(2.4000000000000000000000001))
	fmt.Println(math.Float64bits(2.4) == math.Float64bits(2.4000000000000000000000001))

	s1 := S{ID: 1}
	s2 := S{ID: 1}

	/*
		当 key 是引用类型时，判断两个 key 是否相等，需要 hash 后的值相等并且 key 的字面量相等。
	*/

	var h = map[*S]int{}
	h[&s1] = 1
	h[&s2] = 2
	fmt.Println(h[&s1])
	fmt.Println(h[&s2])
	fmt.Println(s1 == s2)
}
