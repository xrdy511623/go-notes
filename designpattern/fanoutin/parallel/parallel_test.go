package parallel

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestScatter_Basic(t *testing.T) {
	inputs := []int{1, 2, 3, 4, 5}
	results, err := Scatter(context.Background(), inputs, 3, func(_ context.Context, v int) (int, error) {
		return v * 10, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []int{10, 20, 30, 40, 50}
	for i, got := range results {
		if got != want[i] {
			t.Errorf("results[%d] = %d, want %d", i, got, want[i])
		}
	}
}

func TestScatter_Empty(t *testing.T) {
	results, err := Scatter(context.Background(), []int{}, 3, func(_ context.Context, v int) (int, error) {
		return v, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if results != nil {
		t.Fatalf("expected nil for empty input, got %v", results)
	}
}

func TestScatter_PreservesOrder(t *testing.T) {
	inputs := make([]int, 100)
	for i := range inputs {
		inputs[i] = i
	}
	results, err := Scatter(context.Background(), inputs, 8, func(_ context.Context, v int) (string, error) {
		time.Sleep(time.Duration(v%5) * time.Millisecond)
		return fmt.Sprintf("item-%d", v), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range results {
		want := fmt.Sprintf("item-%d", i)
		if r != want {
			t.Errorf("results[%d] = %q, want %q", i, r, want)
		}
	}
}

func TestScatter_BoundedConcurrency(t *testing.T) {
	var running, peak int64
	const limit = 3
	inputs := make([]int, 20)
	for i := range inputs {
		inputs[i] = i
	}

	Scatter(context.Background(), inputs, limit, func(_ context.Context, _ int) (int, error) {
		cur := atomic.AddInt64(&running, 1)
		for {
			old := atomic.LoadInt64(&peak)
			if cur <= old || atomic.CompareAndSwapInt64(&peak, old, cur) {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)
		atomic.AddInt64(&running, -1)
		return 0, nil
	})

	if p := atomic.LoadInt64(&peak); p > limit {
		t.Errorf("peak concurrency %d exceeded limit %d", p, limit)
	}
}

func TestScatter_ErrorCancelsRemaining(t *testing.T) {
	var started int64
	inputs := make([]int, 50)
	for i := range inputs {
		inputs[i] = i
	}

	_, err := Scatter(context.Background(), inputs, 2, func(ctx context.Context, v int) (int, error) {
		atomic.AddInt64(&started, 1)
		if v == 3 {
			return 0, fmt.Errorf("fail at %d", v)
		}
		select {
		case <-time.After(100 * time.Millisecond):
			return v, nil
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	})

	if err == nil {
		t.Fatal("expected error")
	}
	if s := atomic.LoadInt64(&started); s >= int64(len(inputs)) {
		t.Errorf("expected early cancellation, but all %d items were started", s)
	}
}

func TestScatter_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	inputs := make([]int, 10)
	_, err := Scatter(ctx, inputs, 2, func(ctx context.Context, _ int) (int, error) {
		select {
		case <-time.After(1 * time.Second):
			return 1, nil
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	})

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
}

func TestScatter_UnlimitedConcurrency(t *testing.T) {
	var peak int64
	var running int64
	inputs := make([]int, 20)
	for i := range inputs {
		inputs[i] = i
	}

	Scatter(context.Background(), inputs, 0, func(_ context.Context, _ int) (int, error) {
		cur := atomic.AddInt64(&running, 1)
		for {
			old := atomic.LoadInt64(&peak)
			if cur <= old || atomic.CompareAndSwapInt64(&peak, old, cur) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
		atomic.AddInt64(&running, -1)
		return 0, nil
	})

	if p := atomic.LoadInt64(&peak); p < 5 {
		t.Errorf("expected high concurrency with limit=0, peak was only %d", p)
	}
}

func TestForEach_Basic(t *testing.T) {
	var sum int64
	inputs := []int{1, 2, 3, 4, 5}
	err := ForEach(context.Background(), inputs, 3, func(_ context.Context, v int) error {
		atomic.AddInt64(&sum, int64(v))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if sum != 15 {
		t.Fatalf("sum = %d, want 15", sum)
	}
}

func TestForEach_Error(t *testing.T) {
	err := ForEach(context.Background(), []int{1, 2, 3}, 2, func(_ context.Context, v int) error {
		if v == 2 {
			return fmt.Errorf("fail at %d", v)
		}
		return nil
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestForEach_Empty(t *testing.T) {
	err := ForEach(context.Background(), []int{}, 3, func(_ context.Context, _ int) error {
		t.Fatal("should not be called")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// BenchmarkScatter_Sequential processes items sequentially for baseline.
func BenchmarkScatter_Sequential(b *testing.B) {
	inputs := make([]int, 100)
	for i := range inputs {
		inputs[i] = i
	}
	b.ResetTimer()
	for range b.N {
		results := make([]int, len(inputs))
		for i, v := range inputs {
			results[i] = v * v
		}
		_ = results
	}
}

// BenchmarkScatter_Parallel4 processes items with 4 workers.
func BenchmarkScatter_Parallel4(b *testing.B) {
	inputs := make([]int, 100)
	for i := range inputs {
		inputs[i] = i
	}
	b.ResetTimer()
	for range b.N {
		Scatter(context.Background(), inputs, 4, func(_ context.Context, v int) (int, error) {
			return v * v, nil
		})
	}
}
