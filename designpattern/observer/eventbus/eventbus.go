package eventbus

import (
	"fmt"
	"sort"
	"sync"
)

// Event 表示系统中发生的一个事件。
type Event struct {
	Topic   string
	Payload interface{}
}

// Observer 是观察者接口。
// 所有希望接收事件通知的类型都必须实现此接口。
type Observer interface {
	OnEvent(event Event)
	Name() string
}

// EventBus 是观察者模式中的被观察者（Subject）。
// 它维护每个 topic 下的观察者列表，并在事件发生时通知所有相关观察者。
type EventBus struct {
	mu        sync.RWMutex
	observers map[string][]Observer
}

// New 创建一个新的 EventBus 实例。
func New() *EventBus {
	return &EventBus{
		observers: make(map[string][]Observer),
	}
}

// Subscribe 将观察者注册到指定 topic。
// topic 和 obs 不能为空；同一 topic 下不允许重复的观察者名称。
func (eb *EventBus) Subscribe(topic string, obs Observer) error {
	if topic == "" {
		return fmt.Errorf("eventbus: topic must not be empty")
	}
	if obs == nil {
		return fmt.Errorf("eventbus: observer must not be nil")
	}

	eb.mu.Lock()
	defer eb.mu.Unlock()

	for _, existing := range eb.observers[topic] {
		if existing.Name() == obs.Name() {
			return fmt.Errorf("eventbus: observer %q already subscribed to topic %q", obs.Name(), topic)
		}
	}
	eb.observers[topic] = append(eb.observers[topic], obs)
	return nil
}

// Unsubscribe 将指定名称的观察者从 topic 中移除。
func (eb *EventBus) Unsubscribe(topic string, name string) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	observers, ok := eb.observers[topic]
	if !ok {
		return fmt.Errorf("eventbus: topic %q has no observers", topic)
	}

	for i, obs := range observers {
		if obs.Name() == name {
			eb.observers[topic] = append(observers[:i], observers[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("eventbus: observer %q not found for topic %q", name, topic)
}

// Publish 向所有订阅了该 topic 的观察者同步发送事件。
// 先快照观察者列表再释放锁，避免回调中操作 EventBus 导致死锁。
func (eb *EventBus) Publish(event Event) {
	eb.mu.RLock()
	snapshot := make([]Observer, len(eb.observers[event.Topic]))
	copy(snapshot, eb.observers[event.Topic])
	eb.mu.RUnlock()

	for _, obs := range snapshot {
		obs.OnEvent(event)
	}
}

// PublishAsync 向所有订阅者并发发送事件，返回的 channel 在全部回调完成后关闭。
func (eb *EventBus) PublishAsync(event Event) <-chan struct{} {
	eb.mu.RLock()
	snapshot := make([]Observer, len(eb.observers[event.Topic]))
	copy(snapshot, eb.observers[event.Topic])
	eb.mu.RUnlock()

	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(len(snapshot))
	for _, obs := range snapshot {
		go func(o Observer) {
			defer wg.Done()
			o.OnEvent(event)
		}(obs)
	}
	go func() {
		wg.Wait()
		close(done)
	}()
	return done
}

// Topics 返回所有拥有至少一个观察者的 topic（按字典序排列）。
func (eb *EventBus) Topics() []string {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	topics := make([]string, 0, len(eb.observers))
	for topic, obs := range eb.observers {
		if len(obs) > 0 {
			topics = append(topics, topic)
		}
	}
	sort.Strings(topics)
	return topics
}

// Observers 返回指定 topic 下所有观察者的名称（按字典序排列）。
func (eb *EventBus) Observers(topic string) []string {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	names := make([]string, 0, len(eb.observers[topic]))
	for _, obs := range eb.observers[topic] {
		names = append(names, obs.Name())
	}
	sort.Strings(names)
	return names
}
