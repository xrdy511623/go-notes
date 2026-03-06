package eventbus

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

type stubObserver struct {
	name   string
	events []Event
	mu     sync.Mutex
}

func newStub(name string) *stubObserver {
	return &stubObserver{name: name}
}

func (s *stubObserver) OnEvent(event Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
}

func (s *stubObserver) Name() string { return s.name }

func (s *stubObserver) received() []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]Event, len(s.events))
	copy(cp, s.events)
	return cp
}

func TestSubscribe_And_Publish(t *testing.T) {
	bus := New()
	obs := newStub("s1")

	if err := bus.Subscribe("order.created", obs); err != nil {
		t.Fatalf("Subscribe unexpected error: %v", err)
	}

	bus.Publish(Event{Topic: "order.created", Payload: "order-123"})

	events := obs.received()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Payload != "order-123" {
		t.Fatalf("expected payload order-123, got %v", events[0].Payload)
	}
}

func TestPublish_MultipleObservers(t *testing.T) {
	bus := New()
	obs1 := newStub("s1")
	obs2 := newStub("s2")

	bus.Subscribe("topic.a", obs1)
	bus.Subscribe("topic.a", obs2)

	bus.Publish(Event{Topic: "topic.a", Payload: "hello"})

	if len(obs1.received()) != 1 {
		t.Fatal("obs1 should receive 1 event")
	}
	if len(obs2.received()) != 1 {
		t.Fatal("obs2 should receive 1 event")
	}
}

func TestPublish_TopicIsolation(t *testing.T) {
	bus := New()
	obs := newStub("s1")

	bus.Subscribe("topic.a", obs)
	bus.Publish(Event{Topic: "topic.b", Payload: "hello"})

	if len(obs.received()) != 0 {
		t.Fatal("observer should not receive events from a different topic")
	}
}

func TestUnsubscribe_StopsDelivery(t *testing.T) {
	bus := New()
	obs := newStub("s1")

	bus.Subscribe("topic.a", obs)
	bus.Publish(Event{Topic: "topic.a", Payload: "first"})

	if err := bus.Unsubscribe("topic.a", "s1"); err != nil {
		t.Fatalf("Unsubscribe unexpected error: %v", err)
	}

	bus.Publish(Event{Topic: "topic.a", Payload: "second"})

	if len(obs.received()) != 1 {
		t.Fatalf("expected 1 event after unsubscribe, got %d", len(obs.received()))
	}
}

func TestSubscribe_EmptyTopic(t *testing.T) {
	bus := New()
	err := bus.Subscribe("", newStub("s1"))
	if err == nil {
		t.Fatal("expected error for empty topic")
	}
}

func TestSubscribe_NilObserver(t *testing.T) {
	bus := New()
	err := bus.Subscribe("topic.a", nil)
	if err == nil {
		t.Fatal("expected error for nil observer")
	}
}

func TestSubscribe_Duplicate(t *testing.T) {
	bus := New()
	bus.Subscribe("topic.a", newStub("s1"))

	err := bus.Subscribe("topic.a", newStub("s1"))
	if err == nil {
		t.Fatal("expected error for duplicate observer")
	}
}

func TestSubscribe_SameNameDifferentTopic(t *testing.T) {
	bus := New()

	if err := bus.Subscribe("topic.a", newStub("s1")); err != nil {
		t.Fatal(err)
	}
	if err := bus.Subscribe("topic.b", newStub("s1")); err != nil {
		t.Fatal("same observer name on different topics should be allowed")
	}
}

func TestUnsubscribe_UnknownTopic(t *testing.T) {
	bus := New()
	err := bus.Unsubscribe("nonexistent", "s1")
	if err == nil {
		t.Fatal("expected error for unknown topic")
	}
}

func TestUnsubscribe_UnknownObserver(t *testing.T) {
	bus := New()
	bus.Subscribe("topic.a", newStub("s1"))

	err := bus.Unsubscribe("topic.a", "s2")
	if err == nil {
		t.Fatal("expected error for unknown observer")
	}
}

func TestTopics_Sorted(t *testing.T) {
	bus := New()
	bus.Subscribe("order.shipped", newStub("s1"))
	bus.Subscribe("order.created", newStub("s2"))
	bus.Subscribe("user.registered", newStub("s3"))

	topics := bus.Topics()
	expected := []string{"order.created", "order.shipped", "user.registered"}
	if len(topics) != len(expected) {
		t.Fatalf("expected %d topics, got %d", len(expected), len(topics))
	}
	for i, topic := range topics {
		if topic != expected[i] {
			t.Errorf("Topics()[%d] = %q, want %q", i, topic, expected[i])
		}
	}
}

func TestTopics_ExcludesEmpty(t *testing.T) {
	bus := New()
	bus.Subscribe("topic.a", newStub("s1"))
	bus.Unsubscribe("topic.a", "s1")

	topics := bus.Topics()
	if len(topics) != 0 {
		t.Fatalf("expected no topics after all observers removed, got %v", topics)
	}
}

func TestObservers_Sorted(t *testing.T) {
	bus := New()
	bus.Subscribe("topic.a", newStub("charlie"))
	bus.Subscribe("topic.a", newStub("alice"))
	bus.Subscribe("topic.a", newStub("bob"))

	names := bus.Observers("topic.a")
	expected := []string{"alice", "bob", "charlie"}
	if len(names) != len(expected) {
		t.Fatalf("expected %d observers, got %d", len(expected), len(names))
	}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("Observers()[%d] = %q, want %q", i, name, expected[i])
		}
	}
}

func TestObservers_EmptyTopic(t *testing.T) {
	bus := New()
	names := bus.Observers("nonexistent")
	if len(names) != 0 {
		t.Fatalf("expected empty list, got %v", names)
	}
}

func TestPublishAsync(t *testing.T) {
	bus := New()
	obs1 := newStub("s1")
	obs2 := newStub("s2")

	bus.Subscribe("topic.a", obs1)
	bus.Subscribe("topic.a", obs2)

	done := bus.PublishAsync(Event{Topic: "topic.a", Payload: "async"})
	<-done

	if len(obs1.received()) != 1 || len(obs2.received()) != 1 {
		t.Fatal("both observers should receive the async event")
	}
}

func TestPublishAsync_NoObservers(t *testing.T) {
	bus := New()
	done := bus.PublishAsync(Event{Topic: "empty.topic", Payload: "noop"})
	<-done
}

func TestConcurrent_SubscribeAndPublish(t *testing.T) {
	bus := New()
	var count int64

	const n = 100
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			name := fmt.Sprintf("obs-%d", id)
			obs := &countObserver{name: name, count: &count}
			bus.Subscribe("topic.concurrent", obs)
		}(i)
	}
	wg.Wait()

	bus.Publish(Event{Topic: "topic.concurrent", Payload: "ping"})

	if got := atomic.LoadInt64(&count); got != n {
		t.Fatalf("expected %d notifications, got %d", n, got)
	}
}

type countObserver struct {
	name  string
	count *int64
}

func (c *countObserver) OnEvent(_ Event) {
	atomic.AddInt64(c.count, 1)
}

func (c *countObserver) Name() string { return c.name }
