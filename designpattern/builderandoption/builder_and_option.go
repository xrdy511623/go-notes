package builderandoption

import "net/http"

// Config 结构体用于从远程HTTP拉取配置
type Config struct {
	apiKey  string // 鉴权key
	client  *http.Client
	timeout int
	cluster string
}

// Option 是函数选项类型，用于设置Config的属性
type Option func(*Config)

// WithTimeout 函数选项用于设置超时时间
func WithTimeout(timeout int) Option {
	return func(cf *Config) {
		cf.timeout = timeout
	}
}

// WithCluster 函数选项用于设置调用集群
func WithCluster(cluster string) Option {
	return func(cf *Config) {
		cf.cluster = cluster
	}
}

// NewConfig 创建一个新的Config实例，并根据传入的函数选项进行设置
func NewConfig(apiKey string, opts ...Option) (*Config, error) {
	cf := &Config{
		apiKey: apiKey,
		client: &http.Client{},
		// ...
	}

	for _, opt := range opts {
		opt(cf)
	}
	return cf, nil
}
