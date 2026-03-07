package pattern

import (
	"context"
	"sync"
)

// FanOutWorker 启动一个worker，从in读取数据，处理后发送到返回的channel
func FanOutWorker(ctx context.Context, in <-chan int, fn func(int) int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for n := range in {
			select {
			case out <- fn(n):
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// FanOut 启动n个worker并行处理同一个输入channel
func FanOut(ctx context.Context, in <-chan int, workers int, fn func(int) int) []<-chan int {
	outs := make([]<-chan int, workers)
	for i := 0; i < workers; i++ {
		outs[i] = FanOutWorker(ctx, in, fn)
	}
	return outs
}

// FanIn 将多个channel合并为一个channel
func FanIn(ctx context.Context, channels ...<-chan int) <-chan int {
	var wg sync.WaitGroup
	merged := make(chan int)

	for _, ch := range channels {
		wg.Add(1)
		go func(c <-chan int) {
			defer wg.Done()
			for v := range c {
				select {
				case merged <- v:
				case <-ctx.Done():
					return
				}
			}
		}(ch)
	}

	go func() {
		wg.Wait()
		close(merged)
	}()
	return merged
}

// FanOutFanIn 组合fan-out和fan-in：将输入分发给n个worker，结果汇聚到一个channel
func FanOutFanIn(ctx context.Context, in <-chan int, workers int, fn func(int) int) <-chan int {
	outs := FanOut(ctx, in, workers, fn)
	return FanIn(ctx, outs...)
}
