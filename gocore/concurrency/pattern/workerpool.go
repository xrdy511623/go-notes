package pattern

import (
	"context"
	"sync"
)

// WorkerPool 启动固定数量的worker处理jobs channel中的任务
func WorkerPool(ctx context.Context, jobs <-chan int, workers int, fn func(int) int) <-chan int {
	results := make(chan int)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				select {
				case results <- fn(job):
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()
	return results
}

// UnboundedGoroutines 为每个job启动一个新goroutine（对照组，用于性能对比）
func UnboundedGoroutines(ctx context.Context, jobs []int, fn func(int) int) <-chan int {
	results := make(chan int, len(jobs))
	var wg sync.WaitGroup

	for _, job := range jobs {
		wg.Add(1)
		go func(j int) {
			defer wg.Done()
			select {
			case results <- fn(j):
			case <-ctx.Done():
			}
		}(job)
	}

	go func() {
		wg.Wait()
		close(results)
	}()
	return results
}
