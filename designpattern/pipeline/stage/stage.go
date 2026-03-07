package stage

import (
	"context"
	"sync"
)

// Generate sends each item on the returned channel, then closes it.
// The goroutine exits early if ctx is cancelled.
func Generate[T any](ctx context.Context, items ...T) <-chan T {
	out := make(chan T)
	go func() {
		defer close(out)
		for _, item := range items {
			select {
			case out <- item:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// Map transforms each input item using fn and sends the result downstream.
// fn is called synchronously within one goroutine; for CPU-heavy transforms,
// combine with FanOut to distribute work across multiple goroutines.
func Map[In, Out any](ctx context.Context, in <-chan In, fn func(In) Out) <-chan Out {
	out := make(chan Out)
	go func() {
		defer close(out)
		for v := range in {
			select {
			case out <- fn(v):
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// Filter passes through only items satisfying the predicate.
func Filter[T any](ctx context.Context, in <-chan T, fn func(T) bool) <-chan T {
	out := make(chan T)
	go func() {
		defer close(out)
		for v := range in {
			if fn(v) {
				select {
				case out <- v:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out
}

// FanOut starts n goroutines that each read from the same input channel and
// apply fn. Each input item goes to exactly one worker, load-balanced by the
// Go runtime scheduler. Returns n output channels; use Merge to combine them.
func FanOut[In, Out any](ctx context.Context, in <-chan In, n int, fn func(In) Out) []<-chan Out {
	channels := make([]<-chan Out, n)
	for i := range n {
		_ = i
		channels[i] = Map(ctx, in, fn)
	}
	return channels
}

// Merge combines multiple input channels into one output channel (fan-in).
// The output is closed when all inputs are closed or ctx is cancelled.
// Output order is non-deterministic.
func Merge[T any](ctx context.Context, channels ...<-chan T) <-chan T {
	out := make(chan T)
	var wg sync.WaitGroup
	wg.Add(len(channels))
	for _, ch := range channels {
		go func(c <-chan T) {
			defer wg.Done()
			for v := range c {
				select {
				case out <- v:
				case <-ctx.Done():
					return
				}
			}
		}(ch)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

// Take reads at most n items from in.
// Callers should cancel ctx after consuming the output to prevent upstream
// goroutine leaks — this is the standard Go pipeline cancellation pattern.
func Take[T any](ctx context.Context, in <-chan T, n int) <-chan T {
	out := make(chan T)
	go func() {
		defer close(out)
		for i := 0; i < n; i++ {
			select {
			case v, ok := <-in:
				if !ok {
					return
				}
				select {
				case out <- v:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// Collect drains all items from in into a slice. It blocks until in is closed.
func Collect[T any](in <-chan T) []T {
	var result []T
	for v := range in {
		result = append(result, v)
	}
	return result
}
