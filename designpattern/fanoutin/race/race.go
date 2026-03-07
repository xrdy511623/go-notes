package race

import (
	"context"
	"fmt"
)

// First executes all fns concurrently and returns the result from the first
// one that succeeds (returns nil error). Once a winner is found, the remaining
// goroutines are cancelled via context.
//
// If all fns fail, First returns the last error seen.
// If the parent ctx is cancelled before any fn succeeds, First returns ctx.Err().
//
// This is the "hedged request" pattern from Google's "The Tail at Scale" paper:
// send the same request to multiple backends, use the fastest response,
// cancel the rest. It trades redundant work for lower tail latency.
//
// The channel is buffered to len(fns) so that goroutines finishing after
// First returns can send without blocking and be garbage-collected normally.
func First[T any](ctx context.Context, fns ...func(context.Context) (T, error)) (T, error) {
	var zero T
	if len(fns) == 0 {
		return zero, fmt.Errorf("race: no functions provided")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type result struct {
		val T
		err error
	}
	ch := make(chan result, len(fns))

	for _, fn := range fns {
		go func() {
			v, err := fn(ctx)
			select {
			case ch <- result{v, err}:
			case <-ctx.Done():
			}
		}()
	}

	var lastErr error
	for range fns {
		select {
		case r := <-ch:
			if r.err == nil {
				return r.val, nil
			}
			lastErr = r.err
		case <-ctx.Done():
			return zero, ctx.Err()
		}
	}
	return zero, lastErr
}
