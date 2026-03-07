package metricsproxy

import (
	"context"
	"sync/atomic"
	"time"

	"go-notes/designpattern/proxy/fetcher"
)

// Proxy 是度量代理——记录对真实 Fetcher 的调用次数、错误数和总耗时。
//
// 与缓存代理和保护代理不同，度量代理总是转发请求——它更接近装饰器。
// 但在代理模式的分类中，它属于"日志/审计代理"（Logging Proxy）：
// 代理的职责是监控和记录访问行为，而非增强数据处理能力。
//
// 现实中的对应：
//   - Prometheus 的 InstrumentHandler（HTTP 调用计数 + 延迟直方图）
//   - Envoy sidecar（记录所有服务间调用的指标）
//   - database/sql 的 driver wrapper（SQL 慢查询日志）
type Proxy struct {
	real    fetcher.Fetcher
	calls   int64
	errors  int64
	totalNs int64
}

// New 创建一个度量代理。
func New(real fetcher.Fetcher) *Proxy {
	return &Proxy{real: real}
}

func (p *Proxy) Fetch(ctx context.Context, key string) (string, error) {
	start := time.Now()
	value, err := p.real.Fetch(ctx, key)
	elapsed := time.Since(start)

	atomic.AddInt64(&p.calls, 1)
	atomic.AddInt64(&p.totalNs, int64(elapsed))
	if err != nil {
		atomic.AddInt64(&p.errors, 1)
	}
	return value, err
}

// Calls 返回总调用次数。
func (p *Proxy) Calls() int64 { return atomic.LoadInt64(&p.calls) }

// Errors 返回错误次数。
func (p *Proxy) Errors() int64 { return atomic.LoadInt64(&p.errors) }

// AvgLatency 返回平均调用延迟。
func (p *Proxy) AvgLatency() time.Duration {
	calls := atomic.LoadInt64(&p.calls)
	if calls == 0 {
		return 0
	}
	return time.Duration(atomic.LoadInt64(&p.totalNs) / calls)
}
