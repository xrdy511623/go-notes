package resilience

import (
	"context"
	"errors"
	"time"

	"go-notes/designpattern/circuitbreaker/breaker"
	"go-notes/designpattern/circuitbreaker/ratelimiter"
)

// ErrRateLimited 在请求被限流器拒绝时返回。
var ErrRateLimited = errors.New("resilience: rate limited")

// Config 配置弹性调用的三大组件。
//
// 微服务治理的标准执行顺序：
//
//		请求 → [1.限流] → [2.重试循环 → [3.熔断 → fn()]]
//
//	  - 限流在最外层：被限流的请求不计入熔断统计
//	  - 熔断在重试内层：熔断打开时立即返回，不再重试
//	  - 重试只针对瞬时错误：ErrBreakerOpen 不重试（下游已知不可用）
type Config struct {
	Breaker    *breaker.Breaker
	Limiter    *ratelimiter.Limiter
	MaxRetries int                             // 最大尝试次数（含首次调用，默认 1）
	Backoff    func(attempt int) time.Duration // 重试退避策略
}

// Do 用限流 + 熔断 + 重试三板斧执行 fn。
//
// 执行流程：
//  1. 限流检查 — 令牌不足直接返回 ErrRateLimited
//  2. 重试循环 — 最多执行 MaxRetries 次
//  3. 每次循环内通过 Breaker.Do(fn) 执行
//  4. 如果熔断器打开（ErrBreakerOpen），立即返回不再重试
//  5. 如果 fn 返回普通错误，按 Backoff 等待后重试
//  6. Context 取消时立即返回
func Do(ctx context.Context, cfg Config, fn func() error) error {
	if cfg.Limiter != nil && !cfg.Limiter.Allow() {
		return ErrRateLimited
	}

	maxAttempts := cfg.MaxRetries
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	var lastErr error
	for i := range maxAttempts {
		if err := ctx.Err(); err != nil {
			return err
		}

		lastErr = cfg.Breaker.Do(fn)
		if lastErr == nil {
			return nil
		}

		if errors.Is(lastErr, breaker.ErrBreakerOpen) {
			return lastErr
		}

		if i < maxAttempts-1 && cfg.Backoff != nil {
			delay := cfg.Backoff(i)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}
	return lastErr
}
