package breaker

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var errFail = errors.New("service unavailable")

func TestDo_SuccessInClosed(t *testing.T) {
	cb := New(Settings{MaxFailures: 3})
	err := cb.Do(func() error { return nil })
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if cb.State() != StateClosed {
		t.Fatalf("expected closed, got %s", cb.State())
	}
}

func TestDo_FailureInClosed(t *testing.T) {
	cb := New(Settings{MaxFailures: 3})
	err := cb.Do(func() error { return errFail })
	if !errors.Is(err, errFail) {
		t.Fatalf("expected errFail, got %v", err)
	}
	if cb.State() != StateClosed {
		t.Fatalf("expected closed after 1 failure, got %s", cb.State())
	}
}

func TestDo_TripsAfterMaxFailures(t *testing.T) {
	cb := New(Settings{MaxFailures: 3, Timeout: time.Hour})
	for range 3 {
		cb.Do(func() error { return errFail })
	}
	if cb.State() != StateOpen {
		t.Fatalf("expected open after 3 failures, got %s", cb.State())
	}
}

func TestDo_RejectsWhenOpen(t *testing.T) {
	cb := New(Settings{MaxFailures: 1, Timeout: time.Hour})
	cb.Do(func() error { return errFail })

	err := cb.Do(func() error { return nil })
	if !errors.Is(err, ErrBreakerOpen) {
		t.Fatalf("expected ErrBreakerOpen, got %v", err)
	}
}

func TestDo_TransitionsToHalfOpen(t *testing.T) {
	cb := New(Settings{MaxFailures: 1, Timeout: 20 * time.Millisecond})
	cb.Do(func() error { return errFail })

	time.Sleep(30 * time.Millisecond)

	if cb.State() != StateHalfOpen {
		t.Fatalf("expected half-open after timeout, got %s", cb.State())
	}
}

func TestDo_HalfOpenSuccessCloses(t *testing.T) {
	cb := New(Settings{MaxFailures: 1, Timeout: 20 * time.Millisecond})
	cb.Do(func() error { return errFail })
	time.Sleep(30 * time.Millisecond)

	err := cb.Do(func() error { return nil })
	if err != nil {
		t.Fatalf("expected success in half-open, got %v", err)
	}
	if cb.State() != StateClosed {
		t.Fatalf("expected closed after half-open success, got %s", cb.State())
	}
}

func TestDo_HalfOpenFailureOpens(t *testing.T) {
	cb := New(Settings{MaxFailures: 1, Timeout: 20 * time.Millisecond})
	cb.Do(func() error { return errFail })
	time.Sleep(30 * time.Millisecond)

	cb.Do(func() error { return errFail })
	if cb.State() != StateOpen {
		t.Fatalf("expected open after half-open failure, got %s", cb.State())
	}
}

func TestDo_HalfOpenLimitsRequests(t *testing.T) {
	cb := New(Settings{MaxFailures: 1, Timeout: 20 * time.Millisecond, HalfOpenMaxReqs: 1})
	cb.Do(func() error { return errFail })
	time.Sleep(30 * time.Millisecond)

	// First request admitted (HalfOpen)
	done := make(chan struct{})
	go func() {
		cb.Do(func() error {
			<-done // block so the second request hits HalfOpen limit
			return nil
		})
	}()
	time.Sleep(5 * time.Millisecond) // let goroutine enter Do

	err := cb.Do(func() error { return nil })
	close(done)
	if !errors.Is(err, ErrBreakerOpen) {
		t.Fatalf("expected ErrBreakerOpen for excess half-open request, got %v", err)
	}
}

func TestDo_SuccessResetsConsecutiveFailures(t *testing.T) {
	cb := New(Settings{MaxFailures: 3})

	cb.Do(func() error { return errFail })
	cb.Do(func() error { return errFail })
	cb.Do(func() error { return nil }) // reset consecutive failures
	cb.Do(func() error { return errFail })
	cb.Do(func() error { return errFail })

	if cb.State() != StateClosed {
		t.Fatalf("expected closed (success interrupted failures), got %s", cb.State())
	}
	c := cb.Counts()
	if c.ConsecutiveFailures != 2 {
		t.Fatalf("expected 2 consecutive failures, got %d", c.ConsecutiveFailures)
	}
}

func TestDo_OnStateChangeCallback(t *testing.T) {
	var transitions []string
	cb := New(Settings{
		MaxFailures: 1,
		Timeout:     20 * time.Millisecond,
		OnStateChange: func(from, to State) {
			transitions = append(transitions, fmt.Sprintf("%s→%s", from, to))
		},
	})

	cb.Do(func() error { return errFail }) // closed → open
	time.Sleep(30 * time.Millisecond)
	cb.Do(func() error { return nil }) // open → half-open → closed

	expected := []string{"closed→open", "open→half-open", "half-open→closed"}
	if len(transitions) != len(expected) {
		t.Fatalf("expected %d transitions, got %d: %v", len(expected), len(transitions), transitions)
	}
	for i, tr := range transitions {
		if tr != expected[i] {
			t.Errorf("transition[%d] = %q, want %q", i, tr, expected[i])
		}
	}
}

func TestDo_PanicCountsAsFailure(t *testing.T) {
	cb := New(Settings{MaxFailures: 1})

	func() {
		defer func() { recover() }()
		cb.Do(func() error { panic("boom") })
	}()

	if cb.State() != StateOpen {
		t.Fatalf("expected open after panic, got %s", cb.State())
	}
	c := cb.Counts()
	if c.Failures != 1 {
		t.Fatalf("expected 1 failure, got %d", c.Failures)
	}
}

func TestDo_Concurrent(t *testing.T) {
	cb := New(Settings{MaxFailures: 1000})
	var wg sync.WaitGroup
	var successCount int64

	const n = 200
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := cb.Do(func() error { return nil }); err == nil {
				atomic.AddInt64(&successCount, 1)
			}
		}()
	}
	wg.Wait()

	if successCount != n {
		t.Fatalf("expected %d successes, got %d", n, successCount)
	}
	c := cb.Counts()
	if c.Requests != n {
		t.Fatalf("expected %d requests, got %d", n, c.Requests)
	}
}

func TestCounts_TracksRequestStats(t *testing.T) {
	cb := New(Settings{MaxFailures: 10})

	for range 5 {
		cb.Do(func() error { return nil })
	}
	for range 3 {
		cb.Do(func() error { return errFail })
	}

	c := cb.Counts()
	if c.Requests != 8 {
		t.Fatalf("expected 8 requests, got %d", c.Requests)
	}
	if c.Successes != 5 {
		t.Fatalf("expected 5 successes, got %d", c.Successes)
	}
	if c.Failures != 3 {
		t.Fatalf("expected 3 failures, got %d", c.Failures)
	}
	if c.ConsecutiveFailures != 3 {
		t.Fatalf("expected 3 consecutive failures, got %d", c.ConsecutiveFailures)
	}
	if c.ConsecutiveSuccesses != 0 {
		t.Fatalf("expected 0 consecutive successes, got %d", c.ConsecutiveSuccesses)
	}
}

func TestReset_ClearsAll(t *testing.T) {
	cb := New(Settings{MaxFailures: 1, Timeout: time.Hour})
	cb.Do(func() error { return errFail })

	if cb.State() != StateOpen {
		t.Fatalf("expected open, got %s", cb.State())
	}

	cb.Reset()

	if cb.State() != StateClosed {
		t.Fatalf("expected closed after reset, got %s", cb.State())
	}
	c := cb.Counts()
	if c.Requests != 0 || c.Failures != 0 {
		t.Fatalf("expected zeroed counts after reset, got %+v", c)
	}
}

func TestNew_DefaultSettings(t *testing.T) {
	cb := New(Settings{})
	for range 4 {
		cb.Do(func() error { return errFail })
	}
	if cb.State() != StateClosed {
		t.Fatalf("default MaxFailures is 5, should still be closed after 4 failures")
	}
	cb.Do(func() error { return errFail })
	if cb.State() != StateOpen {
		t.Fatalf("expected open after 5 failures (default MaxFailures)")
	}
}

func TestState_String(t *testing.T) {
	tests := []struct {
		s    State
		want string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{State(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.s.String(); got != tt.want {
			t.Errorf("State(%d).String() = %q, want %q", tt.s, got, tt.want)
		}
	}
}

func TestDo_FullCycle(t *testing.T) {
	cb := New(Settings{MaxFailures: 2, Timeout: 20 * time.Millisecond})

	cb.Do(func() error { return errFail })
	cb.Do(func() error { return errFail })
	if cb.State() != StateOpen {
		t.Fatalf("step 1: expected open, got %s", cb.State())
	}

	time.Sleep(30 * time.Millisecond)
	cb.Do(func() error { return errFail })
	if cb.State() != StateOpen {
		t.Fatalf("step 2: expected open after half-open failure, got %s", cb.State())
	}

	time.Sleep(30 * time.Millisecond)
	cb.Do(func() error { return nil })
	if cb.State() != StateClosed {
		t.Fatalf("step 3: expected closed after half-open success, got %s", cb.State())
	}
}
