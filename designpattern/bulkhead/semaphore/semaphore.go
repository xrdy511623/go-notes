package semaphore

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrBulkheadFull 在并发数已达上限且等待超时后返回。
var ErrBulkheadFull = errors.New("bulkhead: concurrency limit reached")

// Bulkhead 使用信号量隔离并发访问，防止单个下游故障耗尽所有资源。
// 内部通过 buffered channel 实现令牌机制：channel 容量 = 最大并发数，
// 发送代表获取令牌，接收代表释放令牌。
type Bulkhead struct {
	name    string
	sem     chan struct{}
	timeout time.Duration
}

// New 创建一个命名的舱壁实例。
// maxConcurrent 为允许的最大并发数，timeout 为获取令牌的最长等待时间。
func New(name string, maxConcurrent int, timeout time.Duration) *Bulkhead {
	if maxConcurrent <= 0 {
		panic(fmt.Sprintf("bulkhead %q: maxConcurrent must be positive, got %d", name, maxConcurrent))
	}
	return &Bulkhead{
		name:    name,
		sem:     make(chan struct{}, maxConcurrent),
		timeout: timeout,
	}
}

// Execute 在舱壁保护下执行 fn。
// 流程：尝试获取令牌 → 执行 fn → 释放令牌。
// 如果并发数已达上限，等待 timeout 或 ctx 取消后返回 ErrBulkheadFull。
// 即使 fn 发生 panic，也会通过 defer 释放令牌，防止泄漏。
func (b *Bulkhead) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	if err := b.acquire(ctx); err != nil {
		return err
	}
	defer b.release()

	return fn(ctx)
}

func (b *Bulkhead) acquire(ctx context.Context) error {
	select {
	case b.sem <- struct{}{}:
		return nil
	default:
	}

	timer := time.NewTimer(b.timeout)
	defer timer.Stop()

	select {
	case b.sem <- struct{}{}:
		return nil
	case <-timer.C:
		return ErrBulkheadFull
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (b *Bulkhead) release() {
	<-b.sem
}

// Name 返回舱壁名称。
func (b *Bulkhead) Name() string { return b.name }

// ActiveCount 返回当前正在执行的请求数。
func (b *Bulkhead) ActiveCount() int { return len(b.sem) }

// MaxConcurrent 返回最大并发数。
func (b *Bulkhead) MaxConcurrent() int { return cap(b.sem) }
