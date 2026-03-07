package performance

import (
	"sync"
	"sync/atomic"
)

// CounterAtomic 使用旧API atomic.AddInt64
type CounterAtomic struct {
	val int64
}

func (c *CounterAtomic) Inc()       { atomic.AddInt64(&c.val, 1) }
func (c *CounterAtomic) Get() int64 { return atomic.LoadInt64(&c.val) }

// CounterAtomicTyped 使用Go 1.19+ 类型化API
type CounterAtomicTyped struct {
	val atomic.Int64
}

func (c *CounterAtomicTyped) Inc()       { c.val.Add(1) }
func (c *CounterAtomicTyped) Get() int64 { return c.val.Load() }

// CounterMutex 使用互斥锁
type CounterMutex struct {
	mu  sync.Mutex
	val int64
}

func (c *CounterMutex) Inc() {
	c.mu.Lock()
	c.val++
	c.mu.Unlock()
}

func (c *CounterMutex) Get() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.val
}

// CounterRWMutex 使用读写锁（读多写少场景）
type CounterRWMutex struct {
	mu  sync.RWMutex
	val int64
}

func (c *CounterRWMutex) Inc() {
	c.mu.Lock()
	c.val++
	c.mu.Unlock()
}

func (c *CounterRWMutex) Get() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.val
}

// ConfigStore 使用atomic.Value存储配置（读多写少典型场景）
type Config struct {
	MaxConn int
	Timeout int
}

type ConfigStoreAtomic struct {
	val atomic.Value
}

func NewConfigStoreAtomic(cfg *Config) *ConfigStoreAtomic {
	s := &ConfigStoreAtomic{}
	s.val.Store(cfg)
	return s
}

func (s *ConfigStoreAtomic) Load() *Config     { return s.val.Load().(*Config) }
func (s *ConfigStoreAtomic) Store(cfg *Config) { s.val.Store(cfg) }

type ConfigStoreMutex struct {
	mu  sync.RWMutex
	cfg *Config
}

func NewConfigStoreMutex(cfg *Config) *ConfigStoreMutex {
	return &ConfigStoreMutex{cfg: cfg}
}

func (s *ConfigStoreMutex) Load() *Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *ConfigStoreMutex) Store(cfg *Config) {
	s.mu.Lock()
	s.cfg = cfg
	s.mu.Unlock()
}
