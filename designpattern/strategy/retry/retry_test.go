package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestConstant(t *testing.T) {
	backoff := Constant(100 * time.Millisecond)
	for _, attempt := range []int{0, 1, 5, 100} {
		if got := backoff(attempt); got != 100*time.Millisecond {
			t.Errorf("Constant(100ms)(%d) = %v, want 100ms", attempt, got)
		}
	}
}

func TestLinear(t *testing.T) {
	backoff := Linear(50 * time.Millisecond)
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 0},
		{1, 50 * time.Millisecond},
		{3, 150 * time.Millisecond},
		{10, 500 * time.Millisecond},
	}
	for _, tt := range tests {
		if got := backoff(tt.attempt); got != tt.want {
			t.Errorf("Linear(50ms)(%d) = %v, want %v", tt.attempt, got, tt.want)
		}
	}
}

func TestExponential(t *testing.T) {
	backoff := Exponential(100*time.Millisecond, 5*time.Second)
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 400 * time.Millisecond},
		{3, 800 * time.Millisecond},
		{10, 5 * time.Second},
	}
	for _, tt := range tests {
		if got := backoff(tt.attempt); got != tt.want {
			t.Errorf("Exponential(100ms,5s)(%d) = %v, want %v", tt.attempt, got, tt.want)
		}
	}
}

func TestExponential_Cap(t *testing.T) {
	backoff := Exponential(1*time.Second, 10*time.Second)
	if got := backoff(20); got != 10*time.Second {
		t.Errorf("expected cap at 10s, got %v", got)
	}
}

func TestWithJitter_Range(t *testing.T) {
	base := Constant(1 * time.Second)
	jittered := WithJitter(base, 0.5)

	seen := make(map[time.Duration]bool)
	for i := 0; i < 100; i++ {
		d := jittered(0)
		seen[d] = true
		if d < 500*time.Millisecond || d > 1500*time.Millisecond {
			t.Errorf("jittered delay %v out of expected range [500ms, 1500ms]", d)
		}
	}
	if len(seen) < 2 {
		t.Error("jitter should produce varying delays")
	}
}

func TestWithJitter_NonNegative(t *testing.T) {
	base := Constant(1 * time.Millisecond)
	jittered := WithJitter(base, 2.0)

	for i := 0; i < 1000; i++ {
		if d := jittered(0); d < 0 {
			t.Fatalf("jittered delay must be non-negative, got %v", d)
		}
	}
}

func TestRetry_SuccessOnFirstAttempt(t *testing.T) {
	calls := 0
	err := Retry(context.Background(), 3, Constant(time.Millisecond), func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestRetry_SuccessOnThirdAttempt(t *testing.T) {
	calls := 0
	err := Retry(context.Background(), 5, Constant(time.Millisecond), func() error {
		calls++
		if calls < 3 {
			return errors.New("transient")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestRetry_AllAttemptsFail(t *testing.T) {
	errPermanent := errors.New("permanent")
	calls := 0
	err := Retry(context.Background(), 3, Constant(time.Millisecond), func() error {
		calls++
		return errPermanent
	})
	if !errors.Is(err, errPermanent) {
		t.Fatalf("expected permanent error, got %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestRetry_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	err := Retry(ctx, 100, Constant(50*time.Millisecond), func() error {
		calls++
		if calls == 2 {
			cancel()
		}
		return errors.New("fail")
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestRetry_SwapStrategy(t *testing.T) {
	strategies := []struct {
		name    string
		backoff BackoffFunc
	}{
		{"constant", Constant(time.Millisecond)},
		{"linear", Linear(time.Millisecond)},
		{"exponential", Exponential(time.Millisecond, 100*time.Millisecond)},
	}
	for _, s := range strategies {
		t.Run(s.name, func(t *testing.T) {
			calls := 0
			err := Retry(context.Background(), 3, s.backoff, func() error {
				calls++
				if calls < 3 {
					return errors.New("retry")
				}
				return nil
			})
			if err != nil {
				t.Fatalf("unexpected error with %s strategy: %v", s.name, err)
			}
		})
	}
}

func TestRetry_SingleAttempt(t *testing.T) {
	calls := 0
	err := Retry(context.Background(), 1, Constant(time.Millisecond), func() error {
		calls++
		return errors.New("fail")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func BenchmarkBackoffFunc(b *testing.B) {
	b.Run("constant", func(b *testing.B) {
		fn := Constant(100 * time.Millisecond)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			fn(i % 10)
		}
	})

	b.Run("exponential", func(b *testing.B) {
		fn := Exponential(100*time.Millisecond, 30*time.Second)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			fn(i % 10)
		}
	})

	b.Run("exponential_with_jitter", func(b *testing.B) {
		fn := WithJitter(Exponential(100*time.Millisecond, 30*time.Second), 0.3)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			fn(i % 10)
		}
	})
}
