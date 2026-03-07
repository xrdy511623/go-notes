package fsm

import (
	"fmt"
	"sort"
	"sync"
)

// State 表示状态机的一个状态。使用类型别名而非裸字符串，防止拼写错误。
type State string

// Event 表示触发状态转换的事件。
type Event string

// Callback 在状态转换时被调用，接收触发事件和转换前后的状态。
//
// 注意：回调在持有 FSM 内部锁时执行。
// 回调内**不能**调用 Fire / Reset 等写方法，否则会死锁。
// 读方法（Current / Is / Can）同样不可调用（Go 的 sync.Mutex 不可重入）。
// 回调已接收到所有需要的信息（event, from, to），无需回查 FSM。
type Callback func(event Event, from, to State)

// Transition 定义一条状态转换规则：在 From 状态收到 Event 后转移到 To。
type Transition struct {
	Event Event
	From  State
	To    State
}

type transKey struct {
	event Event
	from  State
}

// ErrInvalidEvent 表示事件存在，但在当前状态下不允许触发。
var ErrInvalidEvent = fmt.Errorf("fsm: event not valid in current state")

// ErrUnknownEvent 表示事件在整个状态机中未定义。
var ErrUnknownEvent = fmt.Errorf("fsm: unknown event")

// FSM 实现了一个通用的有限状态机。
//
// 设计对标 looplab/fsm，简化了 API：
//   - looplab/fsm 使用 Events 数组 + Callbacks map → 本实现使用 Transition 切片 + 方法注册
//   - looplab/fsm 的 Event 支持多个 Src → 本实现每条 Transition 一个 From（更显式）
//   - 两者都是并发安全的
//
// 标准库中的等价物：
//   - net/http 连接状态（StateNew / StateActive / StateIdle / StateClosed）
//   - bufio.Scanner 内部状态（scanning / done / error）
//   - database/sql.Tx 状态（active / committed / rolledback）
type FSM struct {
	mu          sync.Mutex
	current     State
	transitions map[transKey]State
	allEvents   map[Event]bool
	onEnter     map[State]Callback
	onLeave     map[State]Callback
	onTransit   Callback
}

// ErrDuplicateTransition 表示同一 (Event, From) 组合被重复定义。
var ErrDuplicateTransition = fmt.Errorf("fsm: duplicate transition")

// New 创建一个状态机，初始状态为 initial，转换规则由 transitions 定义。
// 如果存在重复的 (Event, From) 组合，返回 nil 和 ErrDuplicateTransition。
func New(initial State, transitions []Transition) (*FSM, error) {
	f := &FSM{
		current:     initial,
		transitions: make(map[transKey]State, len(transitions)),
		allEvents:   make(map[Event]bool),
		onEnter:     make(map[State]Callback),
		onLeave:     make(map[State]Callback),
	}
	for _, t := range transitions {
		key := transKey{event: t.Event, from: t.From}
		if _, exists := f.transitions[key]; exists {
			return nil, fmt.Errorf("%w: event %q from state %q", ErrDuplicateTransition, t.Event, t.From)
		}
		f.transitions[key] = t.To
		f.allEvents[t.Event] = true
	}
	return f, nil
}

// Fire 触发一个事件，执行对应的状态转换。
//
// 如果事件在当前状态下无效，返回错误（状态不变）。
// 转换成功时，按顺序调用：OnLeave(from) → 更新状态 → OnEnter(to) → OnTransition。
func (f *FSM) Fire(event Event) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	key := transKey{event: event, from: f.current}
	to, ok := f.transitions[key]
	if !ok {
		if f.allEvents[event] {
			return fmt.Errorf("%w: %q in state %q", ErrInvalidEvent, event, f.current)
		}
		return fmt.Errorf("%w: %q", ErrUnknownEvent, event)
	}

	from := f.current

	if cb := f.onLeave[from]; cb != nil {
		cb(event, from, to)
	}
	f.current = to
	if cb := f.onEnter[to]; cb != nil {
		cb(event, from, to)
	}
	if f.onTransit != nil {
		f.onTransit(event, from, to)
	}

	return nil
}

// Current 返回当前状态。
func (f *FSM) Current() State {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.current
}

// Is 判断当前是否处于指定状态。
func (f *FSM) Is(state State) bool {
	return f.Current() == state
}

// Can 判断事件在当前状态下是否可以触发。
func (f *FSM) Can(event Event) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.transitions[transKey{event: event, from: f.current}]
	return ok
}

// AvailableEvents 返回当前状态下所有可触发的事件（按字典序排列）。
func (f *FSM) AvailableEvents() []Event {
	f.mu.Lock()
	defer f.mu.Unlock()

	var events []Event
	for key := range f.transitions {
		if key.from == f.current {
			events = append(events, key.event)
		}
	}
	sort.Slice(events, func(i, j int) bool { return events[i] < events[j] })
	return events
}

// OnEnter 注册进入指定状态时的回调（覆盖之前的注册）。
func (f *FSM) OnEnter(state State, cb Callback) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.onEnter[state] = cb
}

// OnLeave 注册离开指定状态时的回调（覆盖之前的注册）。
func (f *FSM) OnLeave(state State, cb Callback) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.onLeave[state] = cb
}

// OnTransition 注册全局转换回调，每次状态转换都会调用（覆盖之前的注册）。
func (f *FSM) OnTransition(cb Callback) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.onTransit = cb
}

// Reset 将状态机重置到指定状态，不触发任何回调。
func (f *FSM) Reset(state State) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.current = state
}
