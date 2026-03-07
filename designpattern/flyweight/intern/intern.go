package intern

import (
	"strings"
	"sync"
	"sync/atomic"
)

// Interner 是一个并发安全的字符串驻留池。
//
// 相同内容的字符串经过 Get 后返回同一份底层数据，
// 多个调用方共享同一个 string header → 减少堆内存占用和 GC 压力。
//
// 典型场景：日志级别（"error"/"warning"）、HTTP header key、
// Prometheus 标签值——这些字符串大量重复，适合驻留。
//
// 标准库/生态中的等价物：
//   - Go linker 对字符串常量做编译期驻留
//   - net/textproto.CanonicalMIMEHeaderKey 内部缓存规范化 header key
//   - Prometheus/Thanos 使用 interner 去重高基数标签
type Interner struct {
	mu     sync.RWMutex
	pool   map[string]string
	hits   atomic.Int64
	misses atomic.Int64
}

// New 创建一个空的字符串驻留池。
func New() *Interner {
	return &Interner{
		pool: make(map[string]string),
	}
}

// Get 返回 s 的驻留副本。
//
// 如果 s 已存在于池中，直接返回池中的字符串（cache hit）。
// 如果 s 不存在，使用 strings.Clone 创建独立副本后存入池中（cache miss）。
//
// strings.Clone 确保驻留的字符串不会持有调用方大缓冲区的引用——
// 例如 s 是某个 4KB 行的子串，Clone 后只保留实际内容的分配。
func (p *Interner) Get(s string) string {
	p.mu.RLock()
	if canonical, ok := p.pool[s]; ok {
		p.mu.RUnlock()
		p.hits.Add(1)
		return canonical
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()
	if canonical, ok := p.pool[s]; ok {
		p.hits.Add(1)
		return canonical
	}
	cloned := strings.Clone(s)
	p.pool[cloned] = cloned
	p.misses.Add(1)
	return cloned
}

// Len 返回池中驻留的唯一字符串数量。
func (p *Interner) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.pool)
}

// Hits 返回累计缓存命中次数。
func (p *Interner) Hits() int64 {
	return p.hits.Load()
}

// Misses 返回累计缓存未命中次数（即新增驻留次数）。
func (p *Interner) Misses() int64 {
	return p.misses.Load()
}

// Evict 从池中移除指定字符串。如果存在并成功移除返回 true。
func (p *Interner) Evict(s string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.pool[s]; ok {
		delete(p.pool, s)
		return true
	}
	return false
}

// Reset 清空所有驻留条目并重置计数器。
func (p *Interner) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pool = make(map[string]string)
	p.hits.Store(0)
	p.misses.Store(0)
}
