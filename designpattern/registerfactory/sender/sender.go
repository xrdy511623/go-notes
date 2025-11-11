package sender

import (
	"fmt"
	"sync"
)

type Sender interface {
	Send(message string) error
}

type Factory func() Sender

var (
	registry = make(map[string]Factory)
	mu       sync.RWMutex // 读写锁
)

// Register 注册工厂函数(写操作)
func Register(name string, factory Factory) {
	if name == "" || factory == nil {
		fmt.Println("invalid register: name or factory is nil")
		return
	}
	mu.Lock()
	defer mu.Unlock()

	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("sender %s already registered", name))
	}
	registry[name] = factory
}

// New 创建新实例(读操作)
func New(name string) (Sender, error) {
	mu.RLock()
	factory, ok := registry[name]
	mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("sender %s not found", name)
	}
	return factory(), nil
}

// List 列出所有已注册的发送器(读操作)
func List() []string {
	mu.RLock()
	defer mu.RUnlock()

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

func Get(name string) (Sender, error) {
	mu.RLock()
	defer mu.RUnlock()
	if s, ok := registry[name]; ok {
		return s(), nil
	}
	return nil, fmt.Errorf("sender not found: %s", name)
}
