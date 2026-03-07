package cacheproxy

import (
	"context"
	"sync"
	"time"

	"go-notes/designpattern/proxy/fetcher"
)

type entry struct {
	value     string
	expiresAt time.Time
}

// Proxy 是缓存代理——在真实 Fetcher 前增加一层内存缓存。
//
// 代理行为：
//   - 缓存命中 → 直接返回缓存值，**不调用**真实 Fetcher
//   - 缓存未命中/过期 → 调用真实 Fetcher，将结果存入缓存
//
// 这是最典型的代理模式：代理**控制**对真实主体的访问（决定是否调用），
// 而不是像装饰器那样总是转发并增强。
type Proxy struct {
	real  fetcher.Fetcher
	ttl   time.Duration
	mu    sync.RWMutex
	cache map[string]entry
}

// New 创建一个缓存代理。ttl 指定缓存条目的存活时间。
func New(real fetcher.Fetcher, ttl time.Duration) *Proxy {
	return &Proxy{
		real:  real,
		ttl:   ttl,
		cache: make(map[string]entry),
	}
}

func (p *Proxy) Fetch(ctx context.Context, key string) (string, error) {
	p.mu.RLock()
	if e, ok := p.cache[key]; ok && time.Now().Before(e.expiresAt) {
		p.mu.RUnlock()
		return e.value, nil
	}
	p.mu.RUnlock()

	value, err := p.real.Fetch(ctx, key)
	if err != nil {
		return "", err
	}

	p.mu.Lock()
	p.cache[key] = entry{value: value, expiresAt: time.Now().Add(p.ttl)}
	p.mu.Unlock()

	return value, nil
}

// Len 返回当前缓存条目数（含已过期的）。
func (p *Proxy) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.cache)
}
