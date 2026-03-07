package bufferpool

import (
	"bytes"
	"sync"
)

const defaultMaxCap = 64 << 10 // 64KB

// Pool 是 sync.Pool 的类型安全封装，用于复用 *bytes.Buffer。
//
// sync.Pool 是 GC 感知的临时对象池：
//   - 每次 GC 都可能清空池中对象，因此不能用于持久化资源（如数据库连接）
//   - 适合高频分配的临时对象（如序列化缓冲区、HTTP 响应体、日志格式化）
//   - Get 在池为空时调用 New 创建新对象，Put 归还后对象可能被 GC 回收
//
// 本封装在 sync.Pool 之上增加了两个关键实践：
//   - Get 时自动 Reset：防止上一次使用的残留数据泄漏给下一个消费者
//   - Put 时丢弃超大 buffer：防止偶发的大请求导致大 buffer 常驻池中占用内存
type Pool struct {
	pool sync.Pool
}

// New 创建一个 Buffer 对象池。
// initCap 指定每个新 Buffer 的初始容量（字节），避免前几次写入时频繁扩容。
func New(initCap int) *Pool {
	return &Pool{
		pool: sync.Pool{
			New: func() any {
				return bytes.NewBuffer(make([]byte, 0, initCap))
			},
		},
	}
}

// Get 从池中获取一个已重置的 *bytes.Buffer。
// 如果池为空，会通过 New 函数创建新 Buffer。
// 返回的 Buffer 长度为 0，但保留了之前分配的容量。
func (p *Pool) Get() *bytes.Buffer {
	buf := p.pool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// Put 将 Buffer 归还到池中以供复用。
// 容量超过 64KB 的 buffer 会被直接丢弃，防止偶发大请求导致内存膨胀。
func (p *Pool) Put(buf *bytes.Buffer) {
	if buf == nil {
		return
	}
	if buf.Cap() > defaultMaxCap {
		return
	}
	p.pool.Put(buf)
}
