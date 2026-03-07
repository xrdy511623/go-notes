package race

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestFirst_FastestWins(t *testing.T) {
	result, err := First(context.Background(),
		func(ctx context.Context) (string, error) {
			time.Sleep(100 * time.Millisecond)
			return "slow", nil
		},
		func(ctx context.Context) (string, error) {
			time.Sleep(5 * time.Millisecond)
			return "fast", nil
		},
		func(ctx context.Context) (string, error) {
			time.Sleep(50 * time.Millisecond)
			return "medium", nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result != "fast" {
		t.Fatalf("result = %q, want %q", result, "fast")
	}
}

func TestFirst_SkipsErrors(t *testing.T) {
	result, err := First(context.Background(),
		func(ctx context.Context) (string, error) {
			return "", fmt.Errorf("fail-1")
		},
		func(ctx context.Context) (string, error) {
			time.Sleep(10 * time.Millisecond)
			return "success", nil
		},
		func(ctx context.Context) (string, error) {
			return "", fmt.Errorf("fail-2")
		},
	)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if result != "success" {
		t.Fatalf("result = %q, want %q", result, "success")
	}
}

func TestFirst_AllFail(t *testing.T) {
	_, err := First(context.Background(),
		func(ctx context.Context) (int, error) {
			return 0, fmt.Errorf("fail-1")
		},
		func(ctx context.Context) (int, error) {
			return 0, fmt.Errorf("fail-2")
		},
	)
	if err == nil {
		t.Fatal("expected error when all fns fail")
	}
}

func TestFirst_CancelsLosers(t *testing.T) {
	var slowCancelled atomic.Bool
	result, err := First(context.Background(),
		func(ctx context.Context) (string, error) {
			return "instant", nil
		},
		func(ctx context.Context) (string, error) {
			select {
			case <-time.After(5 * time.Second):
				return "slow", nil
			case <-ctx.Done():
				slowCancelled.Store(true)
				return "", ctx.Err()
			}
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result != "instant" {
		t.Fatalf("result = %q, want %q", result, "instant")
	}
	time.Sleep(20 * time.Millisecond)
	if !slowCancelled.Load() {
		t.Error("slow function was not cancelled after winner returned")
	}
}

func TestFirst_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := First(ctx,
		func(ctx context.Context) (int, error) {
			select {
			case <-time.After(1 * time.Second):
				return 1, nil
			case <-ctx.Done():
				return 0, ctx.Err()
			}
		},
	)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
}

func TestFirst_Empty(t *testing.T) {
	_, err := First[int](context.Background())
	if err == nil {
		t.Fatal("expected error for empty fns")
	}
}

func TestFirst_SingleFn(t *testing.T) {
	result, err := First(context.Background(),
		func(ctx context.Context) (string, error) {
			return "only", nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result != "only" {
		t.Fatalf("result = %q, want %q", result, "only")
	}
}
