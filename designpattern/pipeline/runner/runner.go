package runner

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"
)

// StageFunc is a function that runs as one stage of a pipeline.
// It should respect ctx cancellation and return any processing error.
// When a StageFunc returns an error, the Pipeline cancels all other stages.
type StageFunc func(ctx context.Context) error

// Pipeline coordinates the lifecycle of multiple concurrent stages using errgroup.
// Stages communicate through channels passed via closures. If any stage returns
// an error, the errgroup cancels the shared context, causing all stages to exit.
//
// Typical usage:
//
//	ch := make(chan T)
//	p := runner.New()
//	p.Add(sourceStage(ch))     // produces items, closes ch when done
//	p.Add(processingStage(ch)) // reads from ch until closed
//	err := p.Run(ctx)          // blocks until all stages finish
type Pipeline struct {
	stages []StageFunc
}

// New creates an empty Pipeline.
func New() *Pipeline {
	return &Pipeline{}
}

// Add registers a stage to run in the pipeline.
// Stages run concurrently; connect them via channels passed through closures.
func (p *Pipeline) Add(stage StageFunc) *Pipeline {
	p.stages = append(p.stages, stage)
	return p
}

// Run starts all stages in an errgroup and blocks until they all complete.
// If any stage fails, the context is cancelled and Run returns the first error.
// If the caller's ctx is cancelled, all stages observe ctx.Done() and exit.
func (p *Pipeline) Run(ctx context.Context) error {
	if len(p.stages) == 0 {
		return fmt.Errorf("pipeline: no stages to run")
	}
	g, ctx := errgroup.WithContext(ctx)
	for _, s := range p.stages {
		g.Go(func() error { return s(ctx) })
	}
	return g.Wait()
}
