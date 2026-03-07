package stage

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"testing"
	"time"
)

func TestGenerate(t *testing.T) {
	got := Collect(Generate(context.Background(), 1, 2, 3))
	want := []int{1, 2, 3}
	if !sliceEqual(got, want) {
		t.Fatalf("Generate = %v, want %v", got, want)
	}
}

func TestGenerate_Empty(t *testing.T) {
	got := Collect(Generate[int](context.Background()))
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestGenerate_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := Generate(ctx, 1, 2, 3, 4, 5)
	<-ch
	cancel()
	for range ch {
	}
}

func TestMap(t *testing.T) {
	ctx := context.Background()
	got := Collect(Map(ctx, Generate(ctx, 1, 2, 3), func(v int) int { return v * 10 }))
	want := []int{10, 20, 30}
	if !sliceEqual(got, want) {
		t.Fatalf("Map = %v, want %v", got, want)
	}
}

func TestMap_TypeChange(t *testing.T) {
	ctx := context.Background()
	got := Collect(Map(ctx, Generate(ctx, 1, 2, 3), func(v int) string {
		return fmt.Sprintf("item-%d", v)
	}))
	want := []string{"item-1", "item-2", "item-3"}
	if !sliceEqual(got, want) {
		t.Fatalf("Map type change = %v, want %v", got, want)
	}
}

func TestFilter(t *testing.T) {
	ctx := context.Background()
	got := Collect(Filter(ctx, Generate(ctx, 1, 2, 3, 4, 5, 6), func(v int) bool { return v%2 == 0 }))
	want := []int{2, 4, 6}
	if !sliceEqual(got, want) {
		t.Fatalf("Filter = %v, want %v", got, want)
	}
}

func TestFilter_NoneMatch(t *testing.T) {
	ctx := context.Background()
	got := Collect(Filter(ctx, Generate(ctx, 1, 3, 5), func(v int) bool { return v%2 == 0 }))
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestPipeline_Composition(t *testing.T) {
	ctx := context.Background()
	in := Generate(ctx, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
	filtered := Filter(ctx, in, func(v int) bool { return v%2 == 0 })
	mapped := Map(ctx, filtered, func(v int) int { return v * 10 })
	got := Collect(mapped)
	want := []int{20, 40, 60, 80, 100}
	if !sliceEqual(got, want) {
		t.Fatalf("Pipeline = %v, want %v", got, want)
	}
}

func TestMerge(t *testing.T) {
	ctx := context.Background()
	ch1 := Generate(ctx, 1, 2, 3)
	ch2 := Generate(ctx, 4, 5, 6)
	got := Collect(Merge(ctx, ch1, ch2))
	sort.Ints(got)
	want := []int{1, 2, 3, 4, 5, 6}
	if !sliceEqual(got, want) {
		t.Fatalf("Merge = %v, want %v", got, want)
	}
}

func TestMerge_Empty(t *testing.T) {
	got := Collect(Merge[int](context.Background()))
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestFanOut_And_Merge(t *testing.T) {
	ctx := context.Background()
	in := Generate(ctx, 1, 2, 3, 4, 5, 6, 7, 8)
	workers := FanOut(ctx, in, 3, func(v int) int { return v * v })
	got := Collect(Merge(ctx, workers...))
	sort.Ints(got)
	want := []int{1, 4, 9, 16, 25, 36, 49, 64}
	if !sliceEqual(got, want) {
		t.Fatalf("FanOut+Merge = %v, want %v", got, want)
	}
}

func TestTake(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	got := Collect(Take(ctx, Generate(ctx, 1, 2, 3, 4, 5), 3))
	want := []int{1, 2, 3}
	if !sliceEqual(got, want) {
		t.Fatalf("Take = %v, want %v", got, want)
	}
}

func TestTake_MoreThanAvailable(t *testing.T) {
	ctx := context.Background()
	got := Collect(Take(ctx, Generate(ctx, 1, 2), 5))
	want := []int{1, 2}
	if !sliceEqual(got, want) {
		t.Fatalf("Take = %v, want %v", got, want)
	}
}

func TestContextCancel_StopsAllStages(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	before := runtime.NumGoroutine()

	infinite := make(chan int)
	go func() {
		defer close(infinite)
		for i := 0; ; i++ {
			select {
			case infinite <- i:
			case <-ctx.Done():
				return
			}
		}
	}()

	filtered := Filter(ctx, infinite, func(v int) bool { return v%2 == 0 })
	mapped := Map(ctx, filtered, func(v int) int { return v * 10 })

	count := 0
	for v := range mapped {
		_ = v
		count++
		if count >= 5 {
			cancel()
			break
		}
	}
	for range mapped {
	}

	time.Sleep(50 * time.Millisecond)
	after := runtime.NumGoroutine()
	if leaked := after - before; leaked > 2 {
		t.Errorf("potential goroutine leak: before=%d after=%d (delta=%d)", before, after, leaked)
	}
}

// BenchmarkSequential measures raw sequential processing without channels.
func BenchmarkSequential(b *testing.B) {
	items := make([]int, 1000)
	for i := range items {
		items[i] = i
	}
	b.ResetTimer()
	for range b.N {
		var result []int
		for _, v := range items {
			if v%2 == 0 {
				result = append(result, v*10)
			}
		}
		_ = result
	}
}

// BenchmarkPipeline measures channel-based pipeline overhead.
func BenchmarkPipeline(b *testing.B) {
	items := make([]int, 1000)
	for i := range items {
		items[i] = i
	}
	b.ResetTimer()
	for range b.N {
		ctx := context.Background()
		in := Generate(ctx, items...)
		filtered := Filter(ctx, in, func(v int) bool { return v%2 == 0 })
		mapped := Map(ctx, filtered, func(v int) int { return v * 10 })
		_ = Collect(mapped)
	}
}

// BenchmarkFanOut_4Workers measures fan-out parallelism with 4 workers.
func BenchmarkFanOut_4Workers(b *testing.B) {
	items := make([]int, 1000)
	for i := range items {
		items[i] = i
	}
	b.ResetTimer()
	for range b.N {
		ctx := context.Background()
		in := Generate(ctx, items...)
		workers := FanOut(ctx, in, 4, func(v int) int { return v * v })
		_ = Collect(Merge(ctx, workers...))
	}
}

func sliceEqual[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
