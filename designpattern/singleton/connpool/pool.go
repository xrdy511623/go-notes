package connpool

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// Pool 是一个简化的连接池单例，演示饿汉式初始化。
// 包级变量在 import 时即完成初始化，天然并发安全，无需 sync.Once。
type Pool struct {
	mu      sync.Mutex
	conns   []string
	maxSize int
	nextID  int64
}

var instance = newPool(10)

func newPool(maxSize int) *Pool {
	return &Pool{
		conns:   make([]string, 0, maxSize),
		maxSize: maxSize,
	}
}

// Instance 返回全局唯一的连接池实例。
func Instance() *Pool {
	return instance
}

// Get 从池中获取一个连接。如果池为空，创建新连接。
func (p *Pool) Get() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.conns) > 0 {
		conn := p.conns[len(p.conns)-1]
		p.conns = p.conns[:len(p.conns)-1]
		return conn
	}

	id := atomic.AddInt64(&p.nextID, 1)
	return fmt.Sprintf("conn-%d", id)
}

// Put 将连接归还到池中。如果池已满，连接被丢弃。
func (p *Pool) Put(conn string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.conns) < p.maxSize {
		p.conns = append(p.conns, conn)
	}
}

// Len 返回池中当前的空闲连接数。
func (p *Pool) Len() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.conns)
}
