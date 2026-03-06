package builder

import (
	"fmt"
	"time"
)

// Server 表示一个 HTTP 服务器配置。
type Server struct {
	Host           string
	Port           int
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	MaxConnections int
	TLSEnabled     bool
	CertFile       string
	KeyFile        string
}

// Builder 用于分步构建 Server 实例。
// 采用延迟错误策略：首个错误被记录，后续调用直接跳过，最终由 Build() 返回。
type Builder struct {
	server Server
	err    error
}

// NewBuilder 创建一个带默认值的 Builder。host 和 port 是必填参数。
func NewBuilder(host string, port int) *Builder {
	return &Builder{
		server: Server{
			Host:           host,
			Port:           port,
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   30 * time.Second,
			MaxConnections: 1000,
		},
	}
}

// ReadTimeout 设置读超时。
func (b *Builder) ReadTimeout(d time.Duration) *Builder {
	if b.err != nil {
		return b
	}
	if d <= 0 {
		b.err = fmt.Errorf("builder: read timeout must be positive, got %v", d)
		return b
	}
	b.server.ReadTimeout = d
	return b
}

// WriteTimeout 设置写超时。
func (b *Builder) WriteTimeout(d time.Duration) *Builder {
	if b.err != nil {
		return b
	}
	if d <= 0 {
		b.err = fmt.Errorf("builder: write timeout must be positive, got %v", d)
		return b
	}
	b.server.WriteTimeout = d
	return b
}

// MaxConnections 设置最大连接数。
func (b *Builder) MaxConnections(n int) *Builder {
	if b.err != nil {
		return b
	}
	if n <= 0 {
		b.err = fmt.Errorf("builder: max connections must be positive, got %d", n)
		return b
	}
	b.server.MaxConnections = n
	return b
}

// TLS 启用 TLS 并设置证书和密钥文件路径。
func (b *Builder) TLS(certFile, keyFile string) *Builder {
	if b.err != nil {
		return b
	}
	if certFile == "" || keyFile == "" {
		b.err = fmt.Errorf("builder: cert and key files must not be empty")
		return b
	}
	b.server.TLSEnabled = true
	b.server.CertFile = certFile
	b.server.KeyFile = keyFile
	return b
}

// Build 校验配置并返回 Server 实例。
func (b *Builder) Build() (*Server, error) {
	if b.err != nil {
		return nil, b.err
	}
	if b.server.Host == "" {
		return nil, fmt.Errorf("builder: host must not be empty")
	}
	if b.server.Port <= 0 || b.server.Port > 65535 {
		return nil, fmt.Errorf("builder: port must be between 1 and 65535, got %d", b.server.Port)
	}
	srv := b.server
	return &srv, nil
}
