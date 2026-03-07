package sequence

import (
	"iter"
	"slices"
	"testing"
)

func collect[E any](seq iter.Seq[E]) []E {
	var result []E
	for v := range seq {
		result = append(result, v)
	}
	return result
}

func collect2[K, V any](seq iter.Seq2[K, V]) ([]K, []V) {
	var keys []K
	var vals []V
	for k, v := range seq {
		keys = append(keys, k)
		vals = append(vals, v)
	}
	return keys, vals
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

func TestAll(t *testing.T) {
	s := New(3, 1, 4, 1, 5, 9, 2, 6)
	got := collect(s.All())
	assertSliceEqual(t, got, []int{1, 2, 3, 4, 5, 6, 9})
}

func TestAll_Empty(t *testing.T) {
	s := New[int]()
	got := collect(s.All())
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestAll_EarlyBreak(t *testing.T) {
	s := New(1, 2, 3, 4, 5)
	var got []int
	for v := range s.All() {
		got = append(got, v)
		if v == 3 {
			break
		}
	}
	assertSliceEqual(t, got, []int{1, 2, 3})
}

func TestBackward(t *testing.T) {
	s := New(1, 2, 3, 4, 5)
	got := collect(s.Backward())
	assertSliceEqual(t, got, []int{5, 4, 3, 2, 1})
}

func TestBackward_Single(t *testing.T) {
	s := New(42)
	got := collect(s.Backward())
	assertSliceEqual(t, got, []int{42})
}

func TestPairs(t *testing.T) {
	s := New(10, 20, 30)
	keys, vals := collect2(s.Pairs())
	assertSliceEqual(t, keys, []int{0, 1, 2})
	assertSliceEqual(t, vals, []int{10, 20, 30})
}

func TestRange(t *testing.T) {
	s := New(1, 2, 3, 4, 5, 6, 7, 8, 9)
	got := collect(s.Range(3, 7))
	assertSliceEqual(t, got, []int{3, 4, 5, 6})
}

func TestRange_Empty(t *testing.T) {
	s := New(1, 2, 3)
	got := collect(s.Range(5, 10))
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestRange_EarlyBreak(t *testing.T) {
	s := New(1, 2, 3, 4, 5, 6, 7, 8, 9)
	var got []int
	for v := range s.Range(2, 8) {
		got = append(got, v)
		if v == 5 {
			break
		}
	}
	assertSliceEqual(t, got, []int{2, 3, 4, 5})
}

func TestAll_WithPull(t *testing.T) {
	s := New(10, 20, 30, 40, 50)

	next, stop := iter.Pull(s.All())
	defer stop()

	v, ok := next()
	if !ok || v != 10 {
		t.Fatalf("first: got %v/%v, want 10/true", v, ok)
	}
	v, ok = next()
	if !ok || v != 20 {
		t.Fatalf("second: got %v/%v, want 20/true", v, ok)
	}
	stop()
	_, ok = next()
	if ok {
		t.Fatal("expected exhausted after stop")
	}
}

func TestAdd_Deduplication(t *testing.T) {
	s := New(1, 2, 3)
	s.Add(2)
	s.Add(4)
	s.Add(1)
	got := collect(s.All())
	assertSliceEqual(t, got, []int{1, 2, 3, 4})
}

func TestAll_SlicesCollect(t *testing.T) {
	s := New(3, 1, 2)
	got := slices.Collect(s.All())
	assertSliceEqual(t, got, []int{1, 2, 3})
}

func TestAll_String(t *testing.T) {
	s := New("cherry", "apple", "banana")
	got := collect(s.All())
	assertSliceEqual(t, got, []string{"apple", "banana", "cherry"})
}
