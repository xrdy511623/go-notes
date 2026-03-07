package fetcher

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

// Fetcher 从数据源获取数据。
// 代理模式的核心：所有代理和真实主体都实现此接口，
// 调用方无法区分真实主体和代理——这就是代理的透明性。
//
// 与装饰器模式的共同点是"接口一致"，区别在于意图：
//   - 装饰器总是调用被装饰者，增强其行为
//   - 代理可能不调用真实主体（缓存命中、权限拒绝）
type Fetcher interface {
	Fetch(ctx context.Context, key string) (string, error)
}

// SlowFetcher 模拟一个昂贵的数据源（远程 API、数据库查询等）。
// 每次调用都有固定延迟，代表网络或 I/O 开销。
// Calls() 返回实际调用次数，用于验证代理是否正确拦截了请求。
type SlowFetcher struct {
	Latency time.Duration
	calls   int64
}

func (f *SlowFetcher) Fetch(ctx context.Context, key string) (string, error) {
	atomic.AddInt64(&f.calls, 1)
	select {
	case <-time.After(f.Latency):
		return fmt.Sprintf("data-for-%s", key), nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// Calls 返回 Fetch 被调用的次数（并发安全）。
func (f *SlowFetcher) Calls() int64 {
	return atomic.LoadInt64(&f.calls)
}
