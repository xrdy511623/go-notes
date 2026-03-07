package pattern

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"
)

// Task 代表一个可执行的任务
type Task struct {
	ID   string
	Data int
}

// FetchAllBasic 演示errgroup基本用法：首个错误返回，自动取消context
func FetchAllBasic(ctx context.Context, tasks []Task, fn func(ctx context.Context, t Task) error) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, task := range tasks {
		g.Go(func() error {
			return fn(ctx, task)
		})
	}

	return g.Wait() // 返回第一个错误
}

// FetchAllWithLimit 演示errgroup限并发：SetLimit控制最大goroutine数
func FetchAllWithLimit(ctx context.Context, tasks []Task, limit int, fn func(ctx context.Context, t Task) error) error {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(limit)

	for _, task := range tasks {
		g.Go(func() error {
			return fn(ctx, task)
		})
	}

	return g.Wait()
}

// FetchAllCollectErrors 演示收集所有错误（不因单个失败取消全部）
func FetchAllCollectErrors(ctx context.Context, tasks []Task, fn func(ctx context.Context, t Task) error) error {
	g, ctx := errgroup.WithContext(ctx)
	var (
		mu   sync.Mutex
		errs []error
	)

	for _, task := range tasks {
		g.Go(func() error {
			if err := fn(ctx, task); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("task %s: %w", task.ID, err))
				mu.Unlock()
			}
			return nil // 不返回错误，避免触发context取消
		})
	}

	g.Wait()
	return errors.Join(errs...)
}
