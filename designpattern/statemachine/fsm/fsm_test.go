package fsm

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

func mustNew(t *testing.T, initial State, transitions []Transition) *FSM {
	t.Helper()
	m, err := New(initial, transitions)
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}
	return m
}

func turnstile(t *testing.T) *FSM {
	t.Helper()
	return mustNew(t, "locked", []Transition{
		{Event: "coin", From: "locked", To: "unlocked"},
		{Event: "push", From: "unlocked", To: "locked"},
		{Event: "coin", From: "unlocked", To: "unlocked"},
		{Event: "push", From: "locked", To: "locked"},
	})
}

func TestFire_ValidTransition(t *testing.T) {
	m := turnstile(t)
	if err := m.Fire("coin"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Current() != "unlocked" {
		t.Fatalf("expected unlocked, got %s", m.Current())
	}
}

func TestFire_InvalidEvent(t *testing.T) {
	m := mustNew(t, "a", []Transition{
		{Event: "go", From: "a", To: "b"},
		{Event: "back", From: "b", To: "a"},
	})
	err := m.Fire("back")
	if err == nil {
		t.Fatal("expected error for invalid event in current state")
	}
	if !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("expected ErrInvalidEvent, got %v", err)
	}
	if m.Current() != "a" {
		t.Fatal("state should not change on invalid event")
	}
}

func TestFire_UnknownEvent(t *testing.T) {
	m := turnstile(t)
	err := m.Fire("kick")
	if err == nil {
		t.Fatal("expected error for unknown event")
	}
	if !errors.Is(err, ErrUnknownEvent) {
		t.Fatalf("expected ErrUnknownEvent, got %v", err)
	}
}

func TestCurrent_ReturnsInitial(t *testing.T) {
	m := mustNew(t, "idle", nil)
	if m.Current() != "idle" {
		t.Fatalf("expected idle, got %s", m.Current())
	}
}

func TestIs_CurrentState(t *testing.T) {
	m := turnstile(t)
	if !m.Is("locked") {
		t.Fatal("expected Is(locked) == true")
	}
	if m.Is("unlocked") {
		t.Fatal("expected Is(unlocked) == false")
	}
}

func TestCan_ValidEvent(t *testing.T) {
	m := turnstile(t)
	if !m.Can("coin") {
		t.Fatal("expected Can(coin) == true in locked state")
	}
}

func TestCan_InvalidEvent(t *testing.T) {
	m := mustNew(t, "a", []Transition{
		{Event: "go", From: "a", To: "b"},
	})
	if m.Can("go") != true {
		t.Fatal("expected Can(go) == true")
	}
	m.Fire("go")
	if m.Can("go") {
		t.Fatal("expected Can(go) == false in state b")
	}
}

func TestAvailableEvents_Sorted(t *testing.T) {
	m := mustNew(t, "s", []Transition{
		{Event: "z_event", From: "s", To: "s"},
		{Event: "a_event", From: "s", To: "s"},
		{Event: "m_event", From: "s", To: "s"},
	})
	events := m.AvailableEvents()
	expected := []Event{"a_event", "m_event", "z_event"}
	if len(events) != len(expected) {
		t.Fatalf("expected %d events, got %d", len(expected), len(events))
	}
	for i, e := range events {
		if e != expected[i] {
			t.Errorf("events[%d] = %q, want %q", i, e, expected[i])
		}
	}
}

func TestAvailableEvents_OnlyCurrentState(t *testing.T) {
	m := mustNew(t, "a", []Transition{
		{Event: "go", From: "a", To: "b"},
		{Event: "back", From: "b", To: "a"},
	})
	events := m.AvailableEvents()
	if len(events) != 1 || events[0] != "go" {
		t.Fatalf("expected [go], got %v", events)
	}
}

func TestOnEnter_Callback(t *testing.T) {
	m := turnstile(t)
	var entered State
	m.OnEnter("unlocked", func(_ Event, _, to State) {
		entered = to
	})
	m.Fire("coin")
	if entered != "unlocked" {
		t.Fatalf("OnEnter not called, entered = %q", entered)
	}
}

func TestOnLeave_Callback(t *testing.T) {
	m := turnstile(t)
	var left State
	m.OnLeave("locked", func(_ Event, from, _ State) {
		left = from
	})
	m.Fire("coin")
	if left != "locked" {
		t.Fatalf("OnLeave not called, left = %q", left)
	}
}

func TestOnTransition_GlobalCallback(t *testing.T) {
	m := turnstile(t)
	var log []string
	m.OnTransition(func(event Event, from, to State) {
		log = append(log, fmt.Sprintf("%s:%s→%s", event, from, to))
	})

	m.Fire("coin")
	m.Fire("push")

	expected := []string{"coin:locked→unlocked", "push:unlocked→locked"}
	if len(log) != len(expected) {
		t.Fatalf("expected %d transitions, got %d: %v", len(expected), len(log), log)
	}
	for i, entry := range log {
		if entry != expected[i] {
			t.Errorf("log[%d] = %q, want %q", i, entry, expected[i])
		}
	}
}

func TestCallbackOrder(t *testing.T) {
	m := turnstile(t)
	var order []string
	m.OnLeave("locked", func(Event, State, State) { order = append(order, "leave") })
	m.OnEnter("unlocked", func(Event, State, State) { order = append(order, "enter") })
	m.OnTransition(func(Event, State, State) { order = append(order, "transit") })

	m.Fire("coin")

	expected := []string{"leave", "enter", "transit"}
	if len(order) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, order)
	}
	for i, v := range order {
		if v != expected[i] {
			t.Errorf("order[%d] = %q, want %q", i, v, expected[i])
		}
	}
}

func TestFire_SelfTransition(t *testing.T) {
	m := turnstile(t)
	var called bool
	m.OnTransition(func(Event, State, State) { called = true })

	m.Fire("push") // locked → locked (self-transition)
	if m.Current() != "locked" {
		t.Fatal("self-transition should keep same state")
	}
	if !called {
		t.Fatal("callback should fire on self-transition")
	}
}

func TestFire_ChainedTransitions(t *testing.T) {
	m := mustNew(t, "a", []Transition{
		{Event: "next", From: "a", To: "b"},
		{Event: "next", From: "b", To: "c"},
		{Event: "next", From: "c", To: "d"},
	})
	for _, expected := range []State{"b", "c", "d"} {
		m.Fire("next")
		if m.Current() != expected {
			t.Fatalf("expected %s, got %s", expected, m.Current())
		}
	}
}

func TestReset_ChangesState(t *testing.T) {
	m := turnstile(t)
	m.Fire("coin")

	if m.Current() != "unlocked" {
		t.Fatal("expected unlocked before reset")
	}

	m.Reset("locked")
	if m.Current() != "locked" {
		t.Fatalf("expected locked after reset, got %s", m.Current())
	}
}

func TestReset_NoCallbackFired(t *testing.T) {
	m := turnstile(t)
	called := false
	m.OnTransition(func(Event, State, State) { called = true })
	m.OnEnter("locked", func(Event, State, State) { called = true })

	m.Reset("locked")
	if called {
		t.Fatal("Reset should not trigger callbacks")
	}
}

func TestConcurrent_Fire(t *testing.T) {
	m := turnstile(t)
	var successCount int64
	var failCount int64

	var wg sync.WaitGroup
	const n = 200
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := m.Fire("coin"); err != nil {
				atomic.AddInt64(&failCount, 1)
			} else {
				atomic.AddInt64(&successCount, 1)
			}
			if err := m.Fire("push"); err != nil {
				atomic.AddInt64(&failCount, 1)
			} else {
				atomic.AddInt64(&successCount, 1)
			}
		}()
	}
	wg.Wait()

	state := m.Current()
	if state != "locked" && state != "unlocked" {
		t.Fatalf("invalid state after concurrent fire: %s", state)
	}
	total := atomic.LoadInt64(&successCount) + atomic.LoadInt64(&failCount)
	if total != int64(n*2) {
		t.Fatalf("expected %d total Fire calls, got %d", n*2, total)
	}
	if atomic.LoadInt64(&successCount) == 0 {
		t.Fatal("expected at least some successful transitions")
	}
	t.Logf("success=%d fail=%d final_state=%s", successCount, failCount, state)
}

func TestNew_NoTransitions(t *testing.T) {
	m := mustNew(t, "solo", nil)
	if m.Current() != "solo" {
		t.Fatalf("expected solo, got %s", m.Current())
	}
	err := m.Fire("any")
	if !errors.Is(err, ErrUnknownEvent) {
		t.Fatalf("expected ErrUnknownEvent, got %v", err)
	}
}

func TestNew_DuplicateTransition(t *testing.T) {
	_, err := New("a", []Transition{
		{Event: "go", From: "a", To: "b"},
		{Event: "go", From: "a", To: "c"},
	})
	if err == nil {
		t.Fatal("expected error for duplicate transition")
	}
	if !errors.Is(err, ErrDuplicateTransition) {
		t.Fatalf("expected ErrDuplicateTransition, got %v", err)
	}
}
