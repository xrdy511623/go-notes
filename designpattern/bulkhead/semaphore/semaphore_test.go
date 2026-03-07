package semaphore

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestExecute_Success(t *testing.T) {
	bh := New("test", 2, 100*time.Millisecond)
	called := false
	err := bh.Execute(context.Background(), func(_ context.Context) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("fn was not called")
	}
}

func TestExecute_ConcurrencyLimit(t *testing.T) {
	const max = 2
	bh := New("test", max, 50*time.Millisecond)

	blocker := make(chan struct{})
	var running sync.WaitGroup
	running.Add(max)

	for i := 0; i < max; i++ {
		go func() {
			bh.Execute(context.Background(), func(_ context.Context) error {
				running.Done()
				<-blocker
				return nil
			})
		}()
	}

	running.Wait()

	err := bh.Execute(context.Background(), func(_ context.Context) error {
		return nil
	})
	if !errors.Is(err, ErrBulkheadFull) {
		t.Fatalf("expected ErrBulkheadFull, got %v", err)
	}

	close(blocker)
}

func TestExecute_ContextCanceled(t *testing.T) {
	const max = 1
	bh := New("test", max, 5*time.Second)

	blocker := make(chan struct{})
	var running sync.WaitGroup
	running.Add(1)

	go func() {
		bh.Execute(context.Background(), func(_ context.Context) error {
			running.Done()
			<-blocker
			return nil
		})
	}()

	running.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := bh.Execute(ctx, func(_ context.Context) error {
		return nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	close(blocker)
}

func TestExecute_FnError_Propagated(t *testing.T) {
	bh := New("test", 2, 100*time.Millisecond)
	want := errors.New("downstream failure")

	got := bh.Execute(context.Background(), func(_ context.Context) error {
		return want
	})
	if !errors.Is(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestExecute_PanicRecovery(t *testing.T) {
	bh := New("test", 1, 100*time.Millisecond)

	func() {
		defer func() { recover() }()
		bh.Execute(context.Background(), func(_ context.Context) error {
			panic("boom")
		})
	}()

	// 令牌应已归还，新请求可以正常执行
	err := bh.Execute(context.Background(), func(_ context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil after panic recovery, got %v", err)
	}
}

func TestActiveCount_Accurate(t *testing.T) {
	bh := New("test", 5, 100*time.Millisecond)

	if bh.ActiveCount() != 0 {
		t.Fatalf("initial active count = %d, want 0", bh.ActiveCount())
	}

	blocker := make(chan struct{})
	var started sync.WaitGroup
	started.Add(3)

	for i := 0; i < 3; i++ {
		go func() {
			bh.Execute(context.Background(), func(_ context.Context) error {
				started.Done()
				<-blocker
				return nil
			})
		}()
	}

	started.Wait()

	if got := bh.ActiveCount(); got != 3 {
		t.Fatalf("active count = %d, want 3", got)
	}

	close(blocker)
	time.Sleep(20 * time.Millisecond)

	if got := bh.ActiveCount(); got != 0 {
		t.Fatalf("active count after completion = %d, want 0", got)
	}
}

func TestExecute_Sequential_Reuses_Tokens(t *testing.T) {
	bh := New("test", 1, 50*time.Millisecond)

	for i := 0; i < 5; i++ {
		err := bh.Execute(context.Background(), func(_ context.Context) error {
			return nil
		})
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}
	}
}

func TestExecute_Timeout(t *testing.T) {
	bh := New("test", 1, 30*time.Millisecond)

	blocker := make(chan struct{})
	var started sync.WaitGroup
	started.Add(1)

	go func() {
		bh.Execute(context.Background(), func(_ context.Context) error {
			started.Done()
			<-blocker
			return nil
		})
	}()

	started.Wait()

	start := time.Now()
	err := bh.Execute(context.Background(), func(_ context.Context) error {
		return nil
	})
	elapsed := time.Since(start)

	if !errors.Is(err, ErrBulkheadFull) {
		t.Fatalf("expected ErrBulkheadFull, got %v", err)
	}
	if elapsed < 25*time.Millisecond {
		t.Fatalf("returned too quickly (%v), timeout should be ~30ms", elapsed)
	}

	close(blocker)
}

func TestName_And_MaxConcurrent(t *testing.T) {
	bh := New("order-service", 10, time.Second)
	if bh.Name() != "order-service" {
		t.Fatalf("name = %q, want %q", bh.Name(), "order-service")
	}
	if bh.MaxConcurrent() != 10 {
		t.Fatalf("max concurrent = %d, want 10", bh.MaxConcurrent())
	}
}

func TestExecute_Concurrent_Safety(t *testing.T) {
	bh := New("test", 5, 200*time.Millisecond)
	var success int64
	var rejected int64

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			err := bh.Execute(context.Background(), func(_ context.Context) error {
				time.Sleep(50 * time.Millisecond)
				return nil
			})
			if err == nil {
				atomic.AddInt64(&success, 1)
			} else {
				atomic.AddInt64(&rejected, 1)
			}
		}()
	}

	wg.Wait()

	total := atomic.LoadInt64(&success) + atomic.LoadInt64(&rejected)
	if total != goroutines {
		t.Fatalf("total = %d, want %d", total, goroutines)
	}
	if atomic.LoadInt64(&success) == 0 {
		t.Fatal("expected at least some successful executions")
	}
}
