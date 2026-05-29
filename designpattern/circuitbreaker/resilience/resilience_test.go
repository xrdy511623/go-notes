package resilience

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"go-notes/designpattern/circuitbreaker/breaker"
	"go-notes/designpattern/circuitbreaker/ratelimiter"
)

var errTransient = errors.New("transient failure")

func openBreaker() *breaker.Breaker {
	b := breaker.New(breaker.Settings{MaxFailures: 1, Timeout: time.Hour})
	b.Do(func() error { return errors.New("trip") })
	return b
}

func closedBreaker() *breaker.Breaker {
	return breaker.New(breaker.Settings{MaxFailures: 100})
}

func TestDo(t *testing.T) {
	t.Parallel()

	t.Run("happy_path_success_on_first_attempt", func(t *testing.T) {
		t.Parallel()
		var fnCalls atomic.Int32
		err := Do(context.Background(), Config{
			Breaker:    closedBreaker(),
			MaxRetries: 1,
		}, func() error {
			fnCalls.Add(1)
			return nil
		})
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if got := fnCalls.Load(); got != 1 {
			t.Errorf("fn called %d times, want 1", got)
		}
	})

	t.Run("rate_limited_returns_ErrRateLimited_and_fn_not_invoked", func(t *testing.T) {
		t.Parallel()
		lim := ratelimiter.New(0, 0) // zero burst, zero rate → always rejects
		var fnCalls atomic.Int32
		err := Do(context.Background(), Config{
			Breaker:    closedBreaker(),
			Limiter:    lim,
			MaxRetries: 3,
		}, func() error {
			fnCalls.Add(1)
			return nil
		})
		if !errors.Is(err, ErrRateLimited) {
			t.Fatalf("err = %v, want ErrRateLimited", err)
		}
		if got := fnCalls.Load(); got != 0 {
			t.Errorf("fn called %d times when rate limited, want 0", got)
		}
	})

	t.Run("nil_limiter_bypasses_rate_check", func(t *testing.T) {
		t.Parallel()
		err := Do(context.Background(), Config{
			Breaker:    closedBreaker(),
			Limiter:    nil,
			MaxRetries: 1,
		}, func() error { return nil })
		if err != nil {
			t.Fatalf("err = %v, want nil (nil limiter should bypass)", err)
		}
	})

	// KC3 — Defect hypothesis H2: MaxRetries=0 yields zero loop iterations.
	// If maxAttempts normalization `<= 0` is weakened to `< 0`, MaxRetries=0
	// means range(0) which is an empty loop; Do returns nil instead of fn's error.
	t.Run("KC_MaxRetries_zero_defaults_to_one_attempt", func(t *testing.T) {
		t.Parallel()
		var fnCalls atomic.Int32
		err := Do(context.Background(), Config{
			Breaker:    closedBreaker(),
			MaxRetries: 0,
		}, func() error {
			fnCalls.Add(1)
			return errTransient
		})
		if !errors.Is(err, errTransient) {
			t.Fatalf("err = %v, want errTransient; MaxRetries=0 must still execute fn once", err)
		}
		if got := fnCalls.Load(); got != 1 {
			t.Errorf("fn called %d times, want 1", got)
		}
	})

	t.Run("retry_succeeds_on_second_attempt", func(t *testing.T) {
		t.Parallel()
		var fnCalls atomic.Int32
		err := Do(context.Background(), Config{
			Breaker:    closedBreaker(),
			MaxRetries: 3,
			Backoff:    func(int) time.Duration { return 0 },
		}, func() error {
			if fnCalls.Add(1) == 1 {
				return errTransient
			}
			return nil
		})
		if err != nil {
			t.Fatalf("err = %v, want nil (should succeed on retry)", err)
		}
		if got := fnCalls.Load(); got != 2 {
			t.Errorf("fn called %d times, want 2", got)
		}
	})

	t.Run("all_retries_exhausted_returns_last_error", func(t *testing.T) {
		t.Parallel()
		var fnCalls atomic.Int32
		err := Do(context.Background(), Config{
			Breaker:    closedBreaker(),
			MaxRetries: 3,
			Backoff:    func(int) time.Duration { return 0 },
		}, func() error {
			fnCalls.Add(1)
			return errTransient
		})
		if !errors.Is(err, errTransient) {
			t.Fatalf("err = %v, want errTransient after retries exhausted", err)
		}
		if got := fnCalls.Load(); got != 3 {
			t.Errorf("fn called %d times, want 3 (MaxRetries=3)", got)
		}
	})

	// KC1 — Defect hypothesis H3: ErrBreakerOpen is retried instead of
	// returned immediately. If the `errors.Is(lastErr, breaker.ErrBreakerOpen)`
	// guard is removed, Do retries MaxRetries-1 more times, calling backoff
	// each time against a breaker that keeps rejecting.
	t.Run("KC_breaker_open_stops_retries_immediately", func(t *testing.T) {
		t.Parallel()
		var fnCalls atomic.Int32
		var backoffCalls atomic.Int32
		b := openBreaker()
		err := Do(context.Background(), Config{
			Breaker:    b,
			MaxRetries: 5,
			Backoff: func(int) time.Duration {
				backoffCalls.Add(1)
				return 0
			},
		}, func() error {
			fnCalls.Add(1)
			return nil
		})
		if !errors.Is(err, breaker.ErrBreakerOpen) {
			t.Fatalf("err = %v, want ErrBreakerOpen", err)
		}
		if got := backoffCalls.Load(); got != 0 {
			t.Errorf("backoff called %d times, want 0 (ErrBreakerOpen must not trigger retry)", got)
		}
		if got := fnCalls.Load(); got != 0 {
			t.Errorf("fn called %d times, want 0 (breaker rejects before fn)", got)
		}
	})

	t.Run("context_cancelled_before_first_attempt", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		var fnCalls atomic.Int32
		err := Do(ctx, Config{
			Breaker:    closedBreaker(),
			MaxRetries: 3,
		}, func() error {
			fnCalls.Add(1)
			return nil
		})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want context.Canceled", err)
		}
		if got := fnCalls.Load(); got != 0 {
			t.Errorf("fn called %d times with cancelled ctx, want 0", got)
		}
	})

	t.Run("context_deadline_during_backoff_aborts_retry", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		var fnCalls atomic.Int32
		err := Do(ctx, Config{
			Breaker:    closedBreaker(),
			MaxRetries: 3,
			Backoff:    func(int) time.Duration { return 5 * time.Second },
		}, func() error {
			fnCalls.Add(1)
			return errTransient
		})
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("err = %v, want context.DeadlineExceeded during backoff", err)
		}
		if got := fnCalls.Load(); got != 1 {
			t.Errorf("fn called %d times, want 1 (backoff interrupted by deadline)", got)
		}
	})

	t.Run("nil_backoff_retries_without_panic", func(t *testing.T) {
		t.Parallel()
		var fnCalls atomic.Int32
		err := Do(context.Background(), Config{
			Breaker:    closedBreaker(),
			MaxRetries: 3,
			Backoff:    nil,
		}, func() error {
			fnCalls.Add(1)
			return errTransient
		})
		if !errors.Is(err, errTransient) {
			t.Fatalf("err = %v, want errTransient", err)
		}
		if got := fnCalls.Load(); got != 3 {
			t.Errorf("fn called %d times, want 3", got)
		}
	})

	// KC2 — Defect hypothesis H5: backoff applied after last attempt.
	// If boundary `i < maxAttempts-1` is changed to `i < maxAttempts`,
	// backoff sleeps one extra time after the final failed attempt.
	t.Run("KC_backoff_not_called_after_last_attempt", func(t *testing.T) {
		t.Parallel()
		var backoffCalls atomic.Int32
		var fnCalls atomic.Int32
		err := Do(context.Background(), Config{
			Breaker:    closedBreaker(),
			MaxRetries: 2,
			Backoff: func(int) time.Duration {
				backoffCalls.Add(1)
				return 0
			},
		}, func() error {
			fnCalls.Add(1)
			return errTransient
		})
		if !errors.Is(err, errTransient) {
			t.Fatalf("err = %v, want errTransient", err)
		}
		if got := backoffCalls.Load(); got != 1 {
			t.Errorf("backoff called %d times, want 1 (must not backoff after last attempt)", got)
		}
		if got := fnCalls.Load(); got != 2 {
			t.Errorf("fn called %d times, want 2", got)
		}
	})

	t.Run("negative_MaxRetries_defaults_to_one_attempt", func(t *testing.T) {
		t.Parallel()
		var fnCalls atomic.Int32
		err := Do(context.Background(), Config{
			Breaker:    closedBreaker(),
			MaxRetries: -1,
		}, func() error {
			fnCalls.Add(1)
			return errTransient
		})
		if !errors.Is(err, errTransient) {
			t.Fatalf("err = %v, want errTransient", err)
		}
		if got := fnCalls.Load(); got != 1 {
			t.Errorf("fn called %d times, want 1", got)
		}
	})
}
