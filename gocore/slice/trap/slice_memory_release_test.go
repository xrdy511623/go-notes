package main

import (
	"sort"
	"testing"
)

func median(values []uint64) uint64 {
	cp := append([]uint64(nil), values...)
	sort.Slice(cp, func(i, j int) bool {
		return cp[i] < cp[j]
	})
	n := len(cp)
	if n%2 == 1 {
		return cp[n/2]
	}
	return (cp[n/2-1] + cp[n/2]) / 2
}

func TestGetLastBySliceSharesBackingArray(t *testing.T) {
	origin := make([]int, 10, 2048)
	for i := range origin {
		origin[i] = i
	}

	tail := GetLastBySlice(origin)
	if cap(tail) <= len(tail) {
		t.Fatalf("expected retained spare capacity, got len=%d cap=%d", len(tail), cap(tail))
	}

	origin[len(origin)-1] = 9999
	if tail[1] != 9999 {
		t.Fatalf("expected shared backing array, got tail[1]=%d", tail[1])
	}
}

func TestGetLastByCopyDetachesBackingArray(t *testing.T) {
	origin := make([]int, 10, 2048)
	for i := range origin {
		origin[i] = i
	}

	tail := GetLastByCopy(origin)
	if cap(tail) != len(tail) {
		t.Fatalf("expected compact copy slice, got len=%d cap=%d", len(tail), cap(tail))
	}

	origin[len(origin)-1] = 9999
	if tail[1] == 9999 {
		t.Fatal("copy result should not be affected by origin mutation")
	}
}

func TestGetLastRetainedMemoryProfile(t *testing.T) {
	const (
		samples  = 5
		rounds   = 80
		capacity = 128 * 1024 // ~1MB for []int on 64-bit
	)

	sliceRetained := make([]uint64, 0, samples)
	copyRetained := make([]uint64, 0, samples)

	for i := 0; i < samples; i++ {
		sliceRetained = append(sliceRetained, measureRetainedBytes(rounds, capacity, GetLastBySlice))
		copyRetained = append(copyRetained, measureRetainedBytes(rounds, capacity, GetLastByCopy))
	}

	sliceMedian := median(sliceRetained)
	copyMedian := median(copyRetained)

	t.Logf("slice retained median: %.2f MB (samples=%v)", float64(sliceMedian)/1024.0/1024.0, sliceRetained)
	t.Logf("copy retained median: %.2f MB (samples=%v)", float64(copyMedian)/1024.0/1024.0, copyRetained)

	if sliceMedian < 32*1024*1024 {
		t.Fatalf("slice retained memory too low for this scenario: %.2f MB", float64(sliceMedian)/1024.0/1024.0)
	}
	if copyMedian > 8*1024*1024 {
		t.Fatalf("copy retained memory too high for this scenario: %.2f MB", float64(copyMedian)/1024.0/1024.0)
	}
	if sliceMedian < copyMedian*5 {
		t.Fatalf("expected slice retention much higher than copy, got slice=%.2fMB copy=%.2fMB",
			float64(sliceMedian)/1024.0/1024.0, float64(copyMedian)/1024.0/1024.0)
	}
}
