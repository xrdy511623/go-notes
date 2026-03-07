package ratelimiter

import (
	"sync"
	"time"
)

// Limiter 实现令牌桶限流器。
//
// 令牌以固定速率（rate tokens/s）持续填充，最多积攒到 burst 个。
// 每次 Allow() 消耗一个令牌，令牌不足时返回 false。
//
// 这是 golang.org/x/time/rate.Limiter 的简化版本：
//   - x/time/rate 支持 Reserve()（预留令牌）和 Wait()（阻塞等待）
//   - x/time/rate 内部使用 float64 时间差计算，精度更高
//   - x/time/rate 支持 SetLimit/SetBurst 动态调整参数
//
// 本实现只保留 Allow()（非阻塞判断），足够演示令牌桶原理。
//
// 在微服务治理中，限流器通常是最外层的防线：
//
//	请求 → [限流] → [熔断] → [重试] → 下游服务
//
// 限流保护自身不被打爆，熔断保护下游不被雪崩。
type Limiter struct {
	rate     float64 // tokens per second
	burst    float64 // max bucket capacity
	mu       sync.Mutex
	tokens   float64
	lastTime time.Time
}

// New 创建一个令牌桶限流器。
//
//   - rate: 每秒生成的令牌数
//   - burst: 桶容量上限（允许的突发流量）
func New(rate float64, burst int64) *Limiter {
	return &Limiter{
		rate:     rate,
		burst:    float64(burst),
		tokens:   float64(burst),
		lastTime: time.Now(),
	}
}

// Allow 消耗一个令牌，成功返回 true，令牌不足返回 false（非阻塞）。
func (l *Limiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(l.lastTime).Seconds()
	l.tokens += elapsed * l.rate
	if l.tokens > l.burst {
		l.tokens = l.burst
	}
	l.lastTime = now

	if l.tokens >= 1 {
		l.tokens--
		return true
	}
	return false
}

// Tokens 返回当前可用的令牌数（近似值，并发下可能略有偏差）。
func (l *Limiter) Tokens() float64 {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(l.lastTime).Seconds()
	t := l.tokens + elapsed*l.rate
	if t > l.burst {
		t = l.burst
	}
	return t
}
