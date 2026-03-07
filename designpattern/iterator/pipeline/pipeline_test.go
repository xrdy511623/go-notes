package pipeline

import (
	"iter"
	"slices"
	"strings"
	"testing"
)

func fromSlice[E any](s []E) iter.Seq[E] {
	return slices.Values(s)
}

func collect[E any](seq iter.Seq[E]) []E {
	return slices.Collect(seq)
}

func assertSliceEqual[E comparable](t *testing.T, got, want []E) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("length mismatch: got %v (len %d), want %v (len %d)", got, len(got), want, len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("index %d: got %v, want %v", i, got[i], want[i])
		}
	}
}

func TestFilter(t *testing.T) {
	src := fromSlice([]int{1, 2, 3, 4, 5, 6, 7, 8})
	even := Filter(src, func(v int) bool { return v%2 == 0 })
	assertSliceEqual(t, collect(even), []int{2, 4, 6, 8})
}

func TestFilter_Empty(t *testing.T) {
	src := fromSlice([]int{})
	got := collect(Filter(src, func(v int) bool { return true }))
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestFilter_NoneMatch(t *testing.T) {
	src := fromSlice([]int{1, 3, 5, 7})
	got := collect(Filter(src, func(v int) bool { return v%2 == 0 }))
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestFilter_EarlyBreak(t *testing.T) {
	src := fromSlice([]int{1, 2, 3, 4, 5})
	even := Filter(src, func(v int) bool { return v%2 == 0 })
	var got []int
	for v := range even {
		got = append(got, v)
		if v == 4 {
			break
		}
	}
	assertSliceEqual(t, got, []int{2, 4})
}

func TestMap(t *testing.T) {
	src := fromSlice([]string{"hello", "world", "go"})
	upper := Map(src, strings.ToUpper)
	assertSliceEqual(t, collect(upper), []string{"HELLO", "WORLD", "GO"})
}

func TestMap_TypeConversion(t *testing.T) {
	src := fromSlice([]int{1, 2, 3})
	doubled := Map(src, func(v int) int { return v * 2 })
	assertSliceEqual(t, collect(doubled), []int{2, 4, 6})
}

func TestTake(t *testing.T) {
	src := fromSlice([]int{1, 2, 3, 4, 5})
	got := collect(Take(src, 3))
	assertSliceEqual(t, got, []int{1, 2, 3})
}

func TestTake_MoreThanAvailable(t *testing.T) {
	src := fromSlice([]int{1, 2})
	got := collect(Take(src, 10))
	assertSliceEqual(t, got, []int{1, 2})
}

func TestTake_Zero(t *testing.T) {
	src := fromSlice([]int{1, 2, 3})
	got := collect(Take(src, 0))
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestTakeWhile(t *testing.T) {
	src := fromSlice([]int{2, 4, 6, 7, 8, 10})
	got := collect(TakeWhile(src, func(v int) bool { return v%2 == 0 }))
	assertSliceEqual(t, got, []int{2, 4, 6})
}

func TestReduce(t *testing.T) {
	src := fromSlice([]int{1, 2, 3, 4, 5})
	sum := Reduce(src, 0, func(acc, v int) int { return acc + v })
	if sum != 15 {
		t.Fatalf("expected 15, got %d", sum)
	}
}

func TestReduce_Empty(t *testing.T) {
	src := fromSlice([]int{})
	sum := Reduce(src, 42, func(acc, v int) int { return acc + v })
	if sum != 42 {
		t.Fatalf("expected 42 (initial), got %d", sum)
	}
}

func TestConcat(t *testing.T) {
	a := fromSlice([]int{1, 2})
	b := fromSlice([]int{3, 4})
	c := fromSlice([]int{5})
	got := collect(Concat(a, b, c))
	assertSliceEqual(t, got, []int{1, 2, 3, 4, 5})
}

func TestConcat_EarlyBreak(t *testing.T) {
	a := fromSlice([]int{1, 2})
	b := fromSlice([]int{3, 4, 5})
	var got []int
	for v := range Concat(a, b) {
		got = append(got, v)
		if v == 3 {
			break
		}
	}
	assertSliceEqual(t, got, []int{1, 2, 3})
}

func TestZip(t *testing.T) {
	a := fromSlice([]string{"a", "b", "c"})
	b := fromSlice([]int{1, 2, 3})
	var keys []string
	var vals []int
	for k, v := range Zip(a, b) {
		keys = append(keys, k)
		vals = append(vals, v)
	}
	assertSliceEqual(t, keys, []string{"a", "b", "c"})
	assertSliceEqual(t, vals, []int{1, 2, 3})
}

func TestZip_UnequalLength(t *testing.T) {
	a := fromSlice([]int{1, 2, 3, 4, 5})
	b := fromSlice([]int{10, 20})
	var as, bs []int
	for va, vb := range Zip(a, b) {
		as = append(as, va)
		bs = append(bs, vb)
	}
	assertSliceEqual(t, as, []int{1, 2})
	assertSliceEqual(t, bs, []int{10, 20})
}

func TestZip_EarlyBreak(t *testing.T) {
	a := fromSlice([]int{1, 2, 3})
	b := fromSlice([]int{10, 20, 30})
	var as []int
	for va, _ := range Zip(a, b) {
		as = append(as, va)
		if va == 2 {
			break
		}
	}
	assertSliceEqual(t, as, []int{1, 2})
}

func TestPipeline_Composition(t *testing.T) {
	src := fromSlice([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	result := Take(
		Map(
			Filter(src, func(v int) bool { return v%2 == 0 }),
			func(v int) int { return v * 2 },
		),
		3,
	)
	assertSliceEqual(t, collect(result), []int{4, 8, 12})
}

func TestPipeline_ReduceAfterFilter(t *testing.T) {
	src := fromSlice([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	sumOfEven := Reduce(
		Filter(src, func(v int) bool { return v%2 == 0 }),
		0,
		func(acc, v int) int { return acc + v },
	)
	if sumOfEven != 30 {
		t.Fatalf("expected 30, got %d", sumOfEven)
	}
}
