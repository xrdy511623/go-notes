package main

import (
	"math/rand"
	"runtime"
	"testing"
	"time"
)

func generateWithCap(cap int) []int {
	rand.Seed(time.Now().UnixNano())
	nums := make([]int, 0, cap)
	for i := 0; i < cap; i++ {
		nums = append(nums, rand.Int())
	}
	return nums
}

func printMem(t *testing.T) {
	t.Helper()
	var rtm runtime.MemStats
	runtime.ReadMemStats(&rtm)
	t.Logf("%.2f MB", float64(rtm.Alloc)/1024./1024.)
}

func GetLastBySlice(origin []int) []int {
	return origin[len(origin)-2:]
}

func GetLastByCopy(origin []int) []int {
	result := make([]int, 2)
	copy(result, origin[len(origin)-2:])
	return result
}

func testGetLast(t *testing.T, f func([]int) []int) {
	result := make([][]int, 0)
	for k := 0; k < 100; k++ {
		origin := generateWithCap(128 * 1024)
		result = append(result, f(origin))
		// 如果显示开启GC，两者内存占用的差距会更加明显
		// runtime.GC()
	}
	printMem(t)
	_ = result
}
