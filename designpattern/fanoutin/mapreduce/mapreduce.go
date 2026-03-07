package mapreduce

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// MapReduce maps each input item in parallel (fan-out), then reduces the
// intermediate results sequentially (fan-in) into a single output value.
//
// The map phase uses errgroup with bounded concurrency: at most `concurrency`
// goroutines run simultaneously. If any map call returns an error, remaining
// work is cancelled and MapReduce returns the first error.
//
// The reduce phase runs after all map work completes. It processes intermediate
// results in input order, making the output deterministic regardless of
// map execution order.
//
// Modeled after go-zero's mr.MapReduce, but simplified for teaching.
//
//	total, err := MapReduce(ctx, urls, 8,
//	    func(ctx context.Context, url string) (int, error) {
//	        return fetchWordCount(ctx, url)
//	    },
//	    func(acc, count int) int { return acc + count },
//	    0,
//	)
func MapReduce[In, Mid, Out any](
	ctx context.Context,
	inputs []In,
	concurrency int,
	mapFn func(context.Context, In) (Mid, error),
	reduceFn func(Out, Mid) Out,
	initial Out,
) (Out, error) {
	if len(inputs) == 0 {
		return initial, nil
	}
	if concurrency <= 0 {
		concurrency = len(inputs)
	}

	mapped := make([]Mid, len(inputs))
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)

	for i, input := range inputs {
		g.Go(func() error {
			mid, err := mapFn(ctx, input)
			if err != nil {
				return err
			}
			mapped[i] = mid
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return initial, err
	}

	result := initial
	for _, mid := range mapped {
		result = reduceFn(result, mid)
	}
	return result, nil
}
