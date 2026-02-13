package benchmark

// Fib 递归调用求斐波拉契数，其中有大量重复计算，时间复杂度约 O(2^N), 空间复杂度 O(N)
func Fib(n int) int {
	if n < 2 {
		return n
	}
	return Fib(n-2) + Fib(n-1)
}

// FibUseCache 递归调用的同时使用缓存求斐波拉契数
func FibUseCache(n int) int {
	cache := make(map[int]int, n+1)
	var helper func(int) int
	helper = func(n int) int {
		if v, ok := cache[n]; ok {
			return v
		}
		if n < 2 {
			cache[n] = n
			return n
		}
		cache[n] = helper(n-1) + helper(n-2)
		return cache[n]
	}
	return helper(n)
}

// FibUseDynamicProgramming 使用动态规划算法求斐波拉契数，时间复杂度O(N), 空间复杂度O(N)
func FibUseDynamicProgramming(n int) int {
	if n <= 1 {
		return n
	}
	dp := make([]int, n+1)
	dp[1] = 1
	for i := 2; i <= n; i++ {
		dp[i] = dp[i-1] + dp[i-2]
	}
	return dp[n]
}

// FibSimple 在动态规划算法思路基础上进一步降低空间复杂度，时间复杂度O(N), 空间复杂度O(1)
func FibSimple(n int) int {
	if n <= 1 {
		return n
	}
	dp := make([]int, 2)
	dp[1] = 1
	for i := 2; i <= n; i++ {
		sum := dp[0] + dp[1]
		dp[0] = dp[1]
		dp[1] = sum
	}
	return dp[1]
}
