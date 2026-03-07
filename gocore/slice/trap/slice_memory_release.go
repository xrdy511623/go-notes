package main

import (
	"math/rand"
	"runtime"
)

var rng = rand.New(rand.NewSource(1))

func generateWithCap(capacity int) []int {
	nums := make([]int, 0, capacity)
	for i := 0; i < capacity; i++ {
		nums = append(nums, rng.Int())
	}
	return nums
}

func GetLastBySlice(origin []int) []int {
	return origin[len(origin)-2:]
}

func GetLastByCopy(origin []int) []int {
	result := make([]int, 2)
	copy(result, origin[len(origin)-2:])
	return result
}

func currentAllocBytes() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc
}

func measureRetainedBytes(rounds, capacity int, f func([]int) []int) uint64 {
	results := make([][]int, 0, rounds)

	runtime.GC()
	before := currentAllocBytes()

	for k := 0; k < rounds; k++ {
		origin := generateWithCap(capacity)
		results = append(results, f(origin))
	}

	runtime.GC()
	after := currentAllocBytes()
	runtime.KeepAlive(results)

	if after <= before {
		return 0
	}
	return after - before
}
