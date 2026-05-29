package sender

import (
	"errors"
	"fmt"
	"sort"
	"sync"
)

// Sender 是消息发送的统一接口。
type Sender interface {
	Send(message string) error
}

// Closer 可选接口，有资源需释放的 Sender 实现它。
// 对标 http.Flusher、http.Hijacker 的可选接口断言模式。
type Closer interface {
	Close() error
}

// HealthChecker 可选接口，支持健康检查的 Sender 实现它。
type HealthChecker interface {
	HealthCheck() error
}

// Config 是创建 Sender 时传入的配置参数。
// 使用 map[string]string 而非 any：可直接从 JSON/YAML/env 反序列化，
// 类型安全，对标 sql.Open(driverName, dataSourceName) 的精神。
// nil Config 合法，向后兼容无参工厂。
type Config map[string]string

// Factory 是创建 Sender 实例的工厂函数。
// 接受 Config 参数，返回 Sender 和可能的构造错误。
type Factory func(cfg Config) (Sender, error)

var (
	registry = make(map[string]Factory)
	mu       sync.RWMutex
)

// Register 注册一个命名的工厂函数。
// 必须在 init() 阶段调用；name 和 factory 不能为空，重复注册会 panic。
// 设计参考：database/sql.Register、image.RegisterFormat。
func Register(name string, factory Factory) {
	if name == "" {
		panic("sender: Register name must not be empty")
	}
	if factory == nil {
		panic("sender: Register factory must not be nil")
	}

	mu.Lock()
	defer mu.Unlock()

	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("sender: duplicate Register called for %q", name))
	}
	registry[name] = factory
}

// New 通过名称查找工厂并创建一个新的 Sender 实例。
// cfg 可以为 nil，表示使用工厂的默认配置。
func New(name string, cfg Config) (Sender, error) {
	mu.RLock()
	factory, ok := registry[name]
	mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("sender: unknown sender %q (forgotten import?)", name)
	}
	return factory(cfg)
}

// MustNew 与 New 相同，但在查找失败或工厂返回错误时 panic。
// 适用于程序启动阶段，名称已知且不应出错的场景。
// 设计参考：regexp.MustCompile、template.Must。
func MustNew(name string, cfg Config) Sender {
	s, err := New(name, cfg)
	if err != nil {
		panic(err)
	}
	return s
}

// Unregister 从注册表中移除指定名称的工厂。
// 返回 true 表示该名称存在并已删除，false 表示不存在。
// 不会关闭已创建的实例——已创建的 Sender 由调用方管理生命周期。
// 使用场景：热替换、A/B 测试、优雅降级。
func Unregister(name string) bool {
	mu.Lock()
	defer mu.Unlock()

	_, exists := registry[name]
	if exists {
		delete(registry, name)
	}
	return exists
}

// Count 返回当前注册表中的工厂数量。
func Count() int {
	mu.RLock()
	defer mu.RUnlock()
	return len(registry)
}

// List 返回所有已注册的 sender 名称（按字典序排列）。
func List() []string {
	mu.RLock()
	defer mu.RUnlock()

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// CloseAll 关闭一组 Sender 中所有实现了 Closer 的实例。
// 用于优雅关闭。尝试关闭所有 Sender，返回所有错误的聚合（errors.Join）。
func CloseAll(senders []Sender) error {
	var errs []error
	for _, s := range senders {
		if c, ok := s.(Closer); ok {
			if err := c.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

// Reset 清空注册表。仅用于测试。
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	registry = make(map[string]Factory)
}
