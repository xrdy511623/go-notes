package retry

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// BackoffFunc 计算第 attempt 次重试前的等待时间。
// 这就是策略——不同的退避算法只是不同的函数。
//
// 标准库对照：sort.Slice 的 lessFunc、http.Transport 的 RoundTripper。
// 开源对照：cenkalti/backoff 的 BackOff 接口、avast/retry-go 的 DelayTypeFunc。
//
// 为什么用函数类型而不是接口？因为策略只有一个方法，函数类型更轻量，
// 且天然支持闭包捕获配置参数——这是 Go 的杀手锏。
type BackoffFunc func(attempt int) time.Duration

// Constant 返回固定等待时间的退避策略。
// 适用于被调方有固定冷却时间的场景（如 rate-limit 429 响应）。
func Constant(d time.Duration) BackoffFunc {
	return func(_ int) time.Duration {
		return d
	}
}

// Linear 返回线性增长的退避策略：delay = base × attempt。
// attempt=0 时 delay=0，适合首次重试不等待的场景。
func Linear(base time.Duration) BackoffFunc {
	return func(attempt int) time.Duration {
		return base * time.Duration(attempt)
	}
}

// Exponential 返回指数退避策略：delay = base × 2^attempt，上限为 max。
// 这是分布式系统中最常用的退避算法，AWS、GCP 的官方 SDK 都默认使用。
func Exponential(base, max time.Duration) BackoffFunc {
	return func(attempt int) time.Duration {
		d := base * time.Duration(math.Pow(2, float64(attempt)))
		if d > max {
			return max
		}
		return d
	}
}

// WithJitter 在原有策略基础上叠加随机抖动（±factor）。
// 这实际上是 策略装饰器：接收一个 BackoffFunc，返回一个增强版的 BackoffFunc。
//
// 为什么需要 jitter？当大量客户端同时重试时，如果退避时间完全一致，
// 会形成"惊群效应"（thundering herd）。加入随机抖动可以打散重试时间点。
// 参考：AWS Architecture Blog "Exponential Backoff And Jitter"。
func WithJitter(fn BackoffFunc, factor float64) BackoffFunc {
	return func(attempt int) time.Duration {
		d := fn(attempt)
		jitter := time.Duration(float64(d) * factor * (rand.Float64()*2 - 1))
		if result := d + jitter; result > 0 {
			return result
		}
		return 0
	}
}

// Retry 用给定的退避策略执行 fn，直到成功、达到最大次数或 Context 取消。
// maxAttempts 包含首次调用（即 maxAttempts=3 表示最多调用 fn 三次）。
func Retry(ctx context.Context, maxAttempts int, backoff BackoffFunc, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
		}
		if attempt < maxAttempts-1 {
			delay := backoff(attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}
	return lastErr
}
