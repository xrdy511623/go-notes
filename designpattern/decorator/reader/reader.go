package reader

import (
	"io"
	"sync/atomic"
)

// CountingReader 包装 io.Reader，统计累计读取的字节数。
// 这是装饰器模式最纯粹的体现：输入 io.Reader，输出 io.Reader，额外增加计数能力。
// 标准库的 io.LimitReader、io.TeeReader 遵循同样的思路。
type CountingReader struct {
	r     io.Reader
	count int64
}

// NewCountingReader 返回一个包装了 r 的 CountingReader。
func NewCountingReader(r io.Reader) *CountingReader {
	return &CountingReader{r: r}
}

func (c *CountingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	atomic.AddInt64(&c.count, int64(n))
	return n, err
}

// BytesRead 返回累计读取的字节数，并发安全。
func (c *CountingReader) BytesRead() int64 {
	return atomic.LoadInt64(&c.count)
}

// ProgressReader 包装 io.Reader，在每次读取后通过回调上报进度。
// 典型场景：文件上传/下载进度条、大文件处理进度监控。
type ProgressReader struct {
	r          io.Reader
	total      int64
	read       int64
	onProgress func(bytesRead, total int64)
}

// NewProgressReader 返回一个会在每次 Read 后调用 onProgress 的包装 Reader。
// total 为预期总字节数，用于计算百分比；onProgress 为 nil 时退化为透传。
func NewProgressReader(r io.Reader, total int64, onProgress func(bytesRead, total int64)) *ProgressReader {
	return &ProgressReader{r: r, total: total, onProgress: onProgress}
}

func (p *ProgressReader) Read(buf []byte) (int, error) {
	n, err := p.r.Read(buf)
	p.read += int64(n)
	if p.onProgress != nil {
		p.onProgress(p.read, p.total)
	}
	return n, err
}

// BytesRead 返回累计读取的字节数。
func (p *ProgressReader) BytesRead() int64 {
	return p.read
}
