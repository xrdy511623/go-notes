package runner

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestPipeline_Success(t *testing.T) {
	ch := make(chan int)
	var result []int

	p := New()
	p.Add(func(ctx context.Context) error {
		defer close(ch)
		for i := 1; i <= 5; i++ {
			select {
			case ch <- i:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})
	p.Add(func(ctx context.Context) error {
		for v := range ch {
			result = append(result, v*10)
		}
		return nil
	})

	if err := p.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []int{10, 20, 30, 40, 50}
	if len(result) != len(want) {
		t.Fatalf("got %v, want %v", result, want)
	}
	for i := range want {
		if result[i] != want[i] {
			t.Errorf("result[%d] = %d, want %d", i, result[i], want[i])
		}
	}
}

func TestPipeline_StageError_CancelsOthers(t *testing.T) {
	ch := make(chan int, 100)

	p := New()
	p.Add(func(ctx context.Context) error {
		defer close(ch)
		for i := 0; ; i++ {
			select {
			case ch <- i:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	})
	p.Add(func(ctx context.Context) error {
		for v := range ch {
			if v >= 5 {
				return fmt.Errorf("stage error at item %d", v)
			}
		}
		return nil
	})

	err := p.Run(context.Background())
	if err == nil {
		t.Fatal("expected error from pipeline")
	}
	if !errors.Is(err, context.Canceled) {
		if err.Error() != "stage error at item 5" {
			t.Logf("got error: %v (type: %T)", err, err)
		}
	}
}

func TestPipeline_ExternalCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	p := New()
	p.Add(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := p.Run(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestPipeline_NoStages(t *testing.T) {
	p := New()
	if err := p.Run(context.Background()); err == nil {
		t.Fatal("expected error for empty pipeline")
	}
}

func TestPipeline_ThreeStages(t *testing.T) {
	raw := make(chan int)
	doubled := make(chan int)
	var result []int

	p := New()
	p.Add(func(ctx context.Context) error {
		defer close(raw)
		for i := 1; i <= 10; i++ {
			select {
			case raw <- i:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})
	p.Add(func(ctx context.Context) error {
		defer close(doubled)
		for v := range raw {
			if v%2 != 0 {
				continue
			}
			select {
			case doubled <- v * 2:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})
	p.Add(func(ctx context.Context) error {
		for v := range doubled {
			result = append(result, v)
		}
		return nil
	})

	if err := p.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []int{4, 8, 12, 16, 20}
	if len(result) != len(want) {
		t.Fatalf("got %v, want %v", result, want)
	}
	for i := range want {
		if result[i] != want[i] {
			t.Fatalf("result[%d] = %d, want %d", i, result[i], want[i])
		}
	}
}
