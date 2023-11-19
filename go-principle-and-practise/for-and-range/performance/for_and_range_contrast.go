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
