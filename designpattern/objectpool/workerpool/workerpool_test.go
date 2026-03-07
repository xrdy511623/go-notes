package workerpool

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPool_ExecutesAllTasks(t *testing.T) {
	p := New(4)

	var count atomic.Int64
	const n = 100
	var done sync.WaitGroup
	done.Add(n)

	for range n {
		p.Submit(func() {
			defer done.Done()
			count.Add(1)
		})
	}

	done.Wait()
	p.Close()

	if got := count.Load(); got != int64(n) {
		t.Fatalf("expected %d tasks executed, got %d", n, got)
	}
}

func TestPool_BoundsGoroutines(t *testing.T) {
	const workers = 3
	p := New(workers)

	var peak atomic.Int32
	var active atomic.Int32
	var done sync.WaitGroup
	done.Add(20)

	for range 20 {
		p.Submit(func() {
			defer done.Done()
			cur := active.Add(1)
			for {
				old := peak.Load()
				if cur <= old || peak.CompareAndSwap(old, cur) {
					break
				}
			}
			time.Sleep(5 * time.Millisecond)
			active.Add(-1)
		})
	}

	done.Wait()
	p.Close()

	if got := peak.Load(); got > int32(workers) {
		t.Fatalf("peak concurrency %d exceeded %d workers", got, workers)
	}
}

func TestPool_CloseWaitsForCompletion(t *testing.T) {
	p := New(2)

	var completed atomic.Bool
	p.Submit(func() {
		time.Sleep(50 * time.Millisecond)
		completed.Store(true)
	})

	p.Close()

	if !completed.Load() {
		t.Fatal("Close returned before task completed")
	}
}

func TestPool_SingleWorker(t *testing.T) {
	p := New(1)

	var results []int
	var mu sync.Mutex
	var done sync.WaitGroup
	done.Add(5)

	for i := range 5 {
		p.Submit(func() {
			defer done.Done()
			mu.Lock()
			results = append(results, i)
			mu.Unlock()
		})
	}

	done.Wait()
	p.Close()

	if len(results) != 5 {
		t.Fatalf("expected 5 results, got %d", len(results))
	}
}

func TestPool_ConcurrentSubmit(t *testing.T) {
	p := New(4)

	var count atomic.Int64
	const submitters = 10
	const tasksPerSubmitter = 50

	var submittersDone sync.WaitGroup
	var tasksDone sync.WaitGroup
	submittersDone.Add(submitters)
	tasksDone.Add(submitters * tasksPerSubmitter)

	for range submitters {
		go func() {
			defer submittersDone.Done()
			for range tasksPerSubmitter {
				p.Submit(func() {
					defer tasksDone.Done()
					count.Add(1)
				})
			}
		}()
	}

	submittersDone.Wait()
	tasksDone.Wait()
	p.Close()

	expected := int64(submitters * tasksPerSubmitter)
	if got := count.Load(); got != expected {
		t.Fatalf("expected %d tasks, got %d", expected, got)
	}
}

func TestPool_SubmitAfterClose_Panics(t *testing.T) {
	p := New(2)
	p.Close()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on Submit after Close")
		}
	}()

	p.Submit(func() {})
}
