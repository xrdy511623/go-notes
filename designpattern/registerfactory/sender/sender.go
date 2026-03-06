package sender

import (
	"fmt"
	"sort"
	"sync"
)

// Sender 是消息发送的统一接口。
type Sender interface {
	Send(message string) error
}

// Factory 是创建 Sender 实例的工厂函数。
// 每次调用返回一个新的 Sender 实例。
type Factory func() Sender

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
func New(name string) (Sender, error) {
	mu.RLock()
	factory, ok := registry[name]
	mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("sender: unknown sender %q (forgotten import?)", name)
	}
	return factory(), nil
}

// MustNew 与 New 相同，但在查找失败时 panic。
// 适用于程序启动阶段，名称已知且不应出错的场景。
// 设计参考：regexp.MustCompile、template.Must。
func MustNew(name string) Sender {
	s, err := New(name)
	if err != nil {
		panic(err)
	}
	return s
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

// Reset 清空注册表。仅用于测试。
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	registry = make(map[string]Factory)
}
