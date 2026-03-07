package mapreduce

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestMapReduce_Sum(t *testing.T) {
	inputs := []int{1, 2, 3, 4, 5}
	total, err := MapReduce(context.Background(), inputs, 3,
		func(_ context.Context, v int) (int, error) {
			return v * v, nil
		},
		func(acc, v int) int { return acc + v },
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	if total != 55 {
		t.Fatalf("total = %d, want 55 (1+4+9+16+25)", total)
	}
}

func TestMapReduce_WordCount(t *testing.T) {
	texts := []string{"hello world", "go is great", "hello go world"}
	total, err := MapReduce(context.Background(), texts, 2,
		func(_ context.Context, text string) (int, error) {
			return len(strings.Fields(text)), nil
		},
		func(acc, count int) int { return acc + count },
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	if total != 8 {
		t.Fatalf("total = %d, want 8", total)
	}
}

func TestMapReduce_TypeChange(t *testing.T) {
	inputs := []int{1, 2, 3}
	result, err := MapReduce(context.Background(), inputs, 2,
		func(_ context.Context, v int) (string, error) {
			return fmt.Sprintf("%d", v*10), nil
		},
		func(acc, s string) string {
			if acc == "" {
				return s
			}
			return acc + "," + s
		},
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	if result != "10,20,30" {
		t.Fatalf("result = %q, want %q", result, "10,20,30")
	}
}

func TestMapReduce_Empty(t *testing.T) {
	total, err := MapReduce(context.Background(), []int{}, 3,
		func(_ context.Context, v int) (int, error) { return v, nil },
		func(acc, v int) int { return acc + v },
		42,
	)
	if err != nil {
		t.Fatal(err)
	}
	if total != 42 {
		t.Fatalf("empty input should return initial value 42, got %d", total)
	}
}

func TestMapReduce_MapError(t *testing.T) {
	inputs := []int{1, 2, 3, 4, 5}
	_, err := MapReduce(context.Background(), inputs, 2,
		func(_ context.Context, v int) (int, error) {
			if v == 3 {
				return 0, fmt.Errorf("fail at %d", v)
			}
			return v, nil
		},
		func(acc, v int) int { return acc + v },
		0,
	)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMapReduce_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	inputs := make([]int, 10)
	_, err := MapReduce(ctx, inputs, 2,
		func(ctx context.Context, _ int) (int, error) {
			select {
			case <-time.After(1 * time.Second):
				return 1, nil
			case <-ctx.Done():
				return 0, ctx.Err()
			}
		},
		func(acc, v int) int { return acc + v },
		0,
	)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
}

func TestMapReduce_ReduceOrder(t *testing.T) {
	inputs := []int{1, 2, 3, 4, 5}
	result, err := MapReduce(context.Background(), inputs, 5,
		func(_ context.Context, v int) (string, error) {
			time.Sleep(time.Duration(5-v) * time.Millisecond)
			return fmt.Sprintf("%d", v), nil
		},
		func(acc []string, s string) []string { return append(acc, s) },
		[]string{},
	)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"1", "2", "3", "4", "5"}
	for i, v := range result {
		if v != want[i] {
			t.Fatalf("reduce order broken: result[%d] = %q, want %q\nfull result: %v", i, v, want[i], result)
		}
	}
}
