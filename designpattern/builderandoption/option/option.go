package option

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

// Option 是函数选项类型。
// 返回 error 使每个选项可以自行校验参数合法性。
type Option func(*Server) error

// WithReadTimeout 设置读超时。
func WithReadTimeout(d time.Duration) Option {
	return func(s *Server) error {
		if d <= 0 {
			return fmt.Errorf("option: read timeout must be positive, got %v", d)
		}
		s.ReadTimeout = d
		return nil
	}
}

// WithWriteTimeout 设置写超时。
func WithWriteTimeout(d time.Duration) Option {
	return func(s *Server) error {
		if d <= 0 {
			return fmt.Errorf("option: write timeout must be positive, got %v", d)
		}
		s.WriteTimeout = d
		return nil
	}
}

// WithMaxConnections 设置最大连接数。
func WithMaxConnections(n int) Option {
	return func(s *Server) error {
		if n <= 0 {
			return fmt.Errorf("option: max connections must be positive, got %d", n)
		}
		s.MaxConnections = n
		return nil
	}
}

// WithTLS 启用 TLS 并设置证书和密钥文件路径。
func WithTLS(certFile, keyFile string) Option {
	return func(s *Server) error {
		if certFile == "" || keyFile == "" {
			return fmt.Errorf("option: cert and key files must not be empty")
		}
		s.TLSEnabled = true
		s.CertFile = certFile
		s.KeyFile = keyFile
		return nil
	}
}

// NewServer 创建一个 Server 实例。
// host 和 port 是必填参数，opts 是可选的函数选项。
func NewServer(host string, port int, opts ...Option) (*Server, error) {
	if host == "" {
		return nil, fmt.Errorf("option: host must not be empty")
	}
	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("option: port must be between 1 and 65535, got %d", port)
	}

	s := &Server{
		Host:           host,
		Port:           port,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxConnections: 1000,
	}

	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, err
		}
	}
	return s, nil
}
