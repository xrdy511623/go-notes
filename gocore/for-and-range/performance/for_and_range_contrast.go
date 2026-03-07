package performance

import (
	"math/rand"
	"time"
)

func GenerateWithCap(n int) []int {
	rand.Seed(time.Now().UnixNano())
	nums := make([]int, 0, n)
	for i := 0; i < n; i++ {
		nums = append(nums, rand.Int())
	}
	return nums
}

func NormalLoop(s []int) {
	for i := 0; i < len(s); i++ {
		_ = s[i]
	}
}

func EnhanceLoop(s []int) {
	// 假定我们事先不知道切片的长度
	for i, length := 0, len(s); i < length; i++ {
		_ = s[i]
	}
}

type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64
}

// Max 使用泛型来比较两个同类型的值（要求类型是可比较的），并返回较大的值
func Max[T Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}
