package breaker

import (
	"errors"
	"sync"
	"time"
)

// State 表示熔断器的当前状态。
type State int

const (
	StateClosed   State = iota // 正常：请求全部放行
	StateOpen                  // 熔断：请求全部拒绝
	StateHalfOpen              // 探测：放行有限请求，成功则恢复，失败则重新熔断
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Counts 记录熔断器的请求统计。
type Counts struct {
	Requests             int64
	Successes            int64
	Failures             int64
	ConsecutiveSuccesses int64
	ConsecutiveFailures  int64
}

// Settings 配置熔断器的行为参数。
type Settings struct {
	MaxFailures     int64         // 连续失败多少次后触发熔断（默认 5）
	Timeout         time.Duration // Open 状态持续多久后进入 HalfOpen（默认 10s）
	HalfOpenMaxReqs int64         // HalfOpen 状态最多放行多少个探测请求（默认 1）
	OnStateChange   func(from, to State)
}

// ErrBreakerOpen 在熔断器处于 Open/HalfOpen（已满额）状态时返回。
var ErrBreakerOpen = errors.New("breaker: circuit breaker is open")

// Breaker 实现了三态熔断器状态机。
//
// 状态转换：
//
//	Closed ──(连续失败 >= MaxFailures)──→ Open
//	Open ──(超时 Timeout)──→ HalfOpen
//	HalfOpen ──(成功)──→ Closed
//	HalfOpen ──(失败)──→ Open
//
// 对标 sony/gobreaker 的 CircuitBreaker，简化了 ReadyToTrip 和 generation 机制，
// 保留核心状态机逻辑，适合教学和轻量级场景。
type Breaker struct {
	settings     Settings
	mu           sync.Mutex
	state        State
	counts       Counts
	openSince    time.Time
	halfOpenReqs int64
}

// New 创建一个熔断器，使用给定的 Settings（零值字段使用默认值）。
func New(s Settings) *Breaker {
	if s.MaxFailures <= 0 {
		s.MaxFailures = 5
	}
	if s.Timeout <= 0 {
		s.Timeout = 10 * time.Second
	}
	if s.HalfOpenMaxReqs <= 0 {
		s.HalfOpenMaxReqs = 1
	}
	return &Breaker{settings: s}
}

// Do 通过熔断器执行 fn。
//
//   - Closed：直接执行，根据结果更新计数
//   - Open：未超时则返回 ErrBreakerOpen，超时则进入 HalfOpen
//   - HalfOpen：放行有限请求，成功 → Closed，失败 → Open
//
// fn 内的 panic 会被计为一次失败，然后 re-panic（与 sony/gobreaker 行为一致）。
func (b *Breaker) Do(fn func() error) error {
	b.mu.Lock()
	if err := b.beforeRequest(); err != nil {
		b.mu.Unlock()
		return err
	}
	b.mu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			b.afterRequest(false)
			panic(r)
		}
	}()

	err := fn()
	b.afterRequest(err == nil)
	return err
}

// State 返回熔断器的当前有效状态。
// 如果 Open 已超时，返回 HalfOpen（但不触发实际状态转换）。
func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.state == StateOpen && time.Since(b.openSince) >= b.settings.Timeout {
		return StateHalfOpen
	}
	return b.state
}

// Counts 返回当前的请求统计快照。
func (b *Breaker) Counts() Counts {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.counts
}

// Reset 将熔断器重置为 Closed 状态，清空所有计数。
func (b *Breaker) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.state = StateClosed
	b.counts = Counts{}
	b.openSince = time.Time{}
	b.halfOpenReqs = 0
}

func (b *Breaker) beforeRequest() error {
	switch b.state {
	case StateClosed:
		return nil
	case StateOpen:
		if time.Since(b.openSince) < b.settings.Timeout {
			return ErrBreakerOpen
		}
		b.setState(StateHalfOpen)
		b.halfOpenReqs = 1
		return nil
	case StateHalfOpen:
		if b.halfOpenReqs >= b.settings.HalfOpenMaxReqs {
			return ErrBreakerOpen
		}
		b.halfOpenReqs++
		return nil
	}
	return ErrBreakerOpen
}

func (b *Breaker) afterRequest(success bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.counts.Requests++
	if success {
		b.counts.Successes++
		b.counts.ConsecutiveSuccesses++
		b.counts.ConsecutiveFailures = 0
		if b.state == StateHalfOpen {
			b.setState(StateClosed)
		}
	} else {
		b.counts.Failures++
		b.counts.ConsecutiveFailures++
		b.counts.ConsecutiveSuccesses = 0
		switch b.state {
		case StateClosed:
			if b.counts.ConsecutiveFailures >= b.settings.MaxFailures {
				b.setState(StateOpen)
			}
		case StateHalfOpen:
			b.setState(StateOpen)
		}
	}
}

func (b *Breaker) setState(to State) {
	from := b.state
	if from == to {
		return
	}
	b.state = to
	switch to {
	case StateOpen:
		b.openSince = time.Now()
	case StateHalfOpen:
		b.halfOpenReqs = 0
	}
	if b.settings.OnStateChange != nil {
		b.settings.OnStateChange(from, to)
	}
}
