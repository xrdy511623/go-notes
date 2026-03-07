package parallel

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// Scatter executes fn for each input item with bounded concurrency and returns
// all results in input order. If any fn call returns an error, remaining work
// is cancelled via context and Scatter returns the first error.
//
// concurrency controls the maximum number of goroutines running simultaneously.
// A value <= 0 means no limit (one goroutine per input).
//
// Each results[i] is written by exactly one goroutine, so no mutex is needed —
// this is the "index-per-goroutine" pattern safe under the Go memory model
// because errgroup.Wait provides the necessary happens-before edge.
func Scatter[In, Out any](ctx context.Context, inputs []In, concurrency int, fn func(context.Context, In) (Out, error)) ([]Out, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	if concurrency <= 0 {
		concurrency = len(inputs)
	}

	results := make([]Out, len(inputs))
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)

	for i, input := range inputs {
		if ctx.Err() != nil {
			break
		}
		g.Go(func() error {
			out, err := fn(ctx, input)
			if err != nil {
				return err
			}
			results[i] = out
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}

// ForEach executes fn for each input with bounded concurrency.
// Unlike Scatter, it does not collect return values — use it for side effects
// (writes, HTTP calls, notifications) where you only care about success/failure.
func ForEach[T any](ctx context.Context, inputs []T, concurrency int, fn func(context.Context, T) error) error {
	if len(inputs) == 0 {
		return nil
	}
	if concurrency <= 0 {
		concurrency = len(inputs)
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)

	for _, input := range inputs {
		if ctx.Err() != nil {
			break
		}
		g.Go(func() error {
			return fn(ctx, input)
		})
	}

	return g.Wait()
}
