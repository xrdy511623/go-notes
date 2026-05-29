package pool

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSubmit(t *testing.T) {
	t.Parallel()

	t.Run("happy_path_task_executed", func(t *testing.T) {
		t.Parallel()
		wp := NewWorkerPool(2, 10)
		defer wp.Shutdown()

		done := make(chan struct{})
		wp.Submit(func() { close(done) })

		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("task was not executed within timeout")
		}
	})

	t.Run("multiple_tasks_all_executed", func(t *testing.T) {
		t.Parallel()
		const taskCount = 30
		wp := NewWorkerPool(4, 10)

		var count atomic.Int64
		var wg sync.WaitGroup
		wg.Add(taskCount)
		for i := 0; i < taskCount; i++ {
			wp.Submit(func() {
				count.Add(1)
				wg.Done()
			})
		}
		wg.Wait()
		wp.Shutdown()

		if got := count.Load(); got != taskCount {
			t.Fatalf("executed %d tasks, want %d", got, taskCount)
		}
	})

	t.Run("blocks_when_queue_full", func(t *testing.T) {
		t.Parallel()
		wp := NewWorkerPool(1, 1)

		blocker := make(chan struct{})
		started := make(chan struct{})

		wp.Submit(func() {
			close(started)
			<-blocker
		})
		<-started

		// Fill the single queue slot
		wp.Submit(func() {})

		// Next Submit must block since worker is busy and queue is full
		submitted := make(chan struct{})
		go func() {
			wp.Submit(func() {})
			close(submitted)
		}()

		select {
		case <-submitted:
			t.Fatal("Submit returned immediately, want blocking on full queue")
		case <-time.After(100 * time.Millisecond):
		}

		close(blocker)
		<-submitted // wait for blocked Submit to succeed before closing channel
		wp.Shutdown()
	})

	t.Run("killer_all_tasks_executed_under_load", func(t *testing.T) {
		// Killer case — H7: task execution completeness under load
		// Defect hypothesis: off-by-one in worker count or select statement
		// could silently drop tasks
		t.Parallel()
		const (
			workers   = 3
			queueSize = 5
			taskCount = 100
		)
		wp := NewWorkerPool(workers, queueSize)

		var count atomic.Int64
		var wg sync.WaitGroup
		wg.Add(taskCount)
		for i := 0; i < taskCount; i++ {
			wp.Submit(func() {
				count.Add(1)
				wg.Done()
			})
		}
		wg.Wait()
		wp.Shutdown()

		if got := count.Load(); got != taskCount {
			t.Fatalf("executed %d tasks, want %d — tasks were dropped", got, taskCount)
		}
		// If this assertion is removed, a pool that silently drops tasks escapes detection.
	})

	t.Run("concurrent_submits_race_safe", func(t *testing.T) {
		t.Parallel()
		const goroutines = 50
		wp := NewWorkerPool(4, 20)

		var count atomic.Int64
		var wg sync.WaitGroup
		wg.Add(goroutines)

		for i := 0; i < goroutines; i++ {
			go func() {
				defer wg.Done()
				wp.Submit(func() {
					count.Add(1)
				})
			}()
		}
		wg.Wait()
		wp.Shutdown()

		if got := count.Load(); got != goroutines {
			t.Fatalf("executed %d tasks, want %d", got, goroutines)
		}
	})
}

func TestTrySubmit(t *testing.T) {
	t.Parallel()

	t.Run("happy_path_returns_true_and_executes", func(t *testing.T) {
		t.Parallel()
		wp := NewWorkerPool(2, 10)
		defer wp.Shutdown()

		done := make(chan struct{})
		ok := wp.TrySubmit(func() { close(done) })
		if !ok {
			t.Fatal("TrySubmit returned false, want true")
		}

		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("task was not executed within timeout")
		}
	})

	t.Run("killer_queue_full_returns_false", func(t *testing.T) {
		// Killer case — H5: TrySubmit must return false when pool is saturated
		// Defect hypothesis: missing default branch in select, or TrySubmit
		// blocking instead of returning false when queue is full
		t.Parallel()
		wp := NewWorkerPool(1, 1)

		blocker := make(chan struct{})
		started := make(chan struct{})

		wp.Submit(func() {
			close(started)
			<-blocker
		})
		<-started

		if !wp.TrySubmit(func() {}) {
			t.Fatal("first TrySubmit should succeed (queue has 1 free slot)")
		}

		got := wp.TrySubmit(func() {})
		if got {
			t.Fatal("TrySubmit returned true, want false — queue is full")
		}
		// If this assertion is removed, a TrySubmit that blocks forever on a full queue escapes detection.

		close(blocker)
		wp.Shutdown()
	})

	t.Run("returns_true_after_queue_drains", func(t *testing.T) {
		t.Parallel()
		wp := NewWorkerPool(1, 1)

		blocker := make(chan struct{})
		started := make(chan struct{})

		wp.Submit(func() {
			close(started)
			<-blocker
		})
		<-started

		wp.TrySubmit(func() {})

		if wp.TrySubmit(func() {}) {
			t.Fatal("expected false when queue full")
		}

		close(blocker)
		time.Sleep(50 * time.Millisecond)

		done := make(chan struct{})
		if !wp.TrySubmit(func() { close(done) }) {
			t.Fatal("expected true after queue drained")
		}

		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("task was not executed within timeout")
		}
		wp.Shutdown()
	})

	t.Run("concurrent_trysubmit_some_accepted_some_rejected", func(t *testing.T) {
		t.Parallel()
		const goroutines = 50
		wp := NewWorkerPool(2, 5)

		var accepted, rejected atomic.Int64
		var wg sync.WaitGroup
		wg.Add(goroutines)

		for i := 0; i < goroutines; i++ {
			go func() {
				defer wg.Done()
				if wp.TrySubmit(func() {
					time.Sleep(10 * time.Millisecond)
				}) {
					accepted.Add(1)
				} else {
					rejected.Add(1)
				}
			}()
		}

		wg.Wait()
		wp.Shutdown()

		total := accepted.Load() + rejected.Load()
		if total != goroutines {
			t.Fatalf("total = %d, want %d", total, goroutines)
		}
		if accepted.Load() == 0 {
			t.Fatal("expected at least some accepted tasks")
		}
		if rejected.Load() == 0 {
			t.Fatal("expected at least some rejected tasks")
		}
	})

	t.Run("single_slot_queue_boundary", func(t *testing.T) {
		t.Parallel()
		wp := NewWorkerPool(1, 1)
		defer wp.Shutdown()

		done := make(chan struct{})
		ok := wp.TrySubmit(func() { close(done) })
		if !ok {
			t.Fatal("TrySubmit on queue with 1 slot should succeed")
		}

		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("task was not executed")
		}
	})
}

func TestShutdown(t *testing.T) {
	t.Parallel()

	t.Run("happy_path_clean_shutdown", func(t *testing.T) {
		t.Parallel()
		wp := NewWorkerPool(3, 10)
		done := make(chan struct{})
		wp.Submit(func() { close(done) })
		<-done
		wp.Shutdown()
	})

	t.Run("killer_buffered_tasks_drain_before_return", func(t *testing.T) {
		// Killer case — H3: Shutdown must drain all buffered tasks
		// Defect hypothesis: closing the tasks channel might cause workers to
		// return from select before processing remaining buffered items
		t.Parallel()
		const taskCount = 20
		wp := NewWorkerPool(2, taskCount)

		var count atomic.Int64
		blocker := make(chan struct{})
		started := make(chan struct{}, 2)

		for i := 0; i < 2; i++ {
			wp.Submit(func() {
				started <- struct{}{}
				<-blocker
			})
		}
		<-started
		<-started

		for i := 0; i < taskCount; i++ {
			wp.Submit(func() {
				count.Add(1)
			})
		}

		close(blocker)
		wp.Shutdown()

		if got := count.Load(); got != int64(taskCount) {
			t.Fatalf("drained %d tasks, want %d — buffered tasks were dropped", got, taskCount)
		}
		// If this assertion is removed, a Shutdown that drops buffered tasks escapes detection.
	})

	t.Run("idempotent_multiple_calls_no_panic", func(t *testing.T) {
		t.Parallel()
		wp := NewWorkerPool(2, 5)
		wp.Submit(func() {})

		wp.Shutdown()
		wp.Shutdown()
		wp.Shutdown()
	})

	t.Run("concurrent_shutdown_no_panic", func(t *testing.T) {
		t.Parallel()
		wp := NewWorkerPool(4, 10)
		for i := 0; i < 10; i++ {
			wp.Submit(func() {
				time.Sleep(5 * time.Millisecond)
			})
		}

		var wg sync.WaitGroup
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				wp.Shutdown()
			}()
		}
		wg.Wait()
	})

	t.Run("empty_pool_immediate_shutdown", func(t *testing.T) {
		t.Parallel()
		wp := NewWorkerPool(2, 5)
		start := time.Now()
		wp.Shutdown()
		elapsed := time.Since(start)
		if elapsed > 500*time.Millisecond {
			t.Fatalf("Shutdown took %v on empty pool, expected near-immediate", elapsed)
		}
	})
}
