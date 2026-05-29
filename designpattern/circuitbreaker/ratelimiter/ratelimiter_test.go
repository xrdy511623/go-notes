package ratelimiter

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("happy_path_initializes_full_bucket", func(t *testing.T) {
		t.Parallel()
		lim := New(10, 5)

		for i := range 5 {
			if !lim.Allow() {
				t.Fatalf("Allow() #%d = false, want true (initial burst)", i+1)
			}
		}
		if lim.Allow() {
			t.Fatalf("Allow() #6 = true, want false (burst exhausted)")
		}
	})

	t.Run("burst_one_single_token", func(t *testing.T) {
		t.Parallel()
		lim := New(0, 1)

		if !lim.Allow() {
			t.Fatalf("Allow() #1 = false, want true")
		}
		if lim.Allow() {
			t.Fatalf("Allow() #2 = true, want false (single token consumed)")
		}
	})

	t.Run("killer/burst_zero_initializes_empty", func(t *testing.T) {
		t.Parallel()
		// H3: if New sets tokens to nonzero when burst=0, Allow succeeds incorrectly.
		lim := New(10, 0)

		if lim.Allow() {
			t.Fatalf("Allow() = true with burst=0, want false; bucket must start empty")
		}
		tok := lim.Tokens()
		if tok > 0.1 {
			t.Errorf("Tokens() = %f with burst=0, want ≈ 0", tok)
		}
	})
}

func TestAllow(t *testing.T) {
	t.Parallel()

	t.Run("happy_path_consumes_burst_then_blocks", func(t *testing.T) {
		t.Parallel()
		lim := New(0, 5) // rate=0 eliminates time dependency

		for i := range 5 {
			if !lim.Allow() {
				t.Fatalf("Allow() #%d = false, want true", i+1)
			}
		}
		if lim.Allow() {
			t.Fatalf("Allow() #6 = true, want false (burst exhausted)")
		}
	})

	t.Run("burst_zero_always_rejects", func(t *testing.T) {
		t.Parallel()
		lim := New(100, 0)

		for i := range 3 {
			if lim.Allow() {
				t.Fatalf("Allow() call #%d = true with burst=0, want false", i+1)
			}
		}
	})

	t.Run("burst_one_exactly_one_allowed", func(t *testing.T) {
		t.Parallel()
		lim := New(0, 1)

		if !lim.Allow() {
			t.Fatalf("Allow() #1 = false, want true")
		}
		if lim.Allow() {
			t.Fatalf("Allow() #2 = true, want false")
		}
	})

	t.Run("rate_zero_no_refill_after_exhaustion", func(t *testing.T) {
		t.Parallel()
		lim := New(0, 3)
		for range 3 {
			lim.Allow()
		}
		time.Sleep(50 * time.Millisecond)

		if lim.Allow() {
			t.Fatalf("Allow() = true with rate=0 after exhaustion, want false")
		}
	})

	t.Run("refill_allows_after_sufficient_wait", func(t *testing.T) {
		t.Parallel()
		lim := New(100, 1) // 100 tokens/s, burst=1

		if !lim.Allow() {
			t.Fatalf("initial Allow() = false, want true")
		}
		time.Sleep(50 * time.Millisecond) // 100*0.05 = 5 tokens, capped to burst=1

		if !lim.Allow() {
			t.Fatalf("Allow() after refill = false, want true")
		}
	})

	t.Run("tokens_capped_at_burst_after_long_wait", func(t *testing.T) {
		t.Parallel()
		lim := New(100, 3)
		for range 3 {
			lim.Allow()
		}
		time.Sleep(100 * time.Millisecond) // 100*0.1 = 10 tokens, capped at burst=3

		got := lim.Tokens()
		if got > 3.1 {
			t.Errorf("Tokens() = %f after long wait, want <= 3.0 (burst cap)", got)
		}
		if got < 2.5 {
			t.Errorf("Tokens() = %f after long wait, want ≈ 3.0", got)
		}
	})

	t.Run("killer/fractional_token_below_one_must_reject", func(t *testing.T) {
		t.Parallel()
		// H2: if Allow() checks tokens > 0 instead of tokens >= 1,
		// fractional tokens (e.g., 0.1) would incorrectly pass.
		lim := New(1, 1) // 1 token/s, burst=1

		if !lim.Allow() {
			t.Fatalf("initial Allow() = false, want true")
		}
		time.Sleep(100 * time.Millisecond) // 1*0.1 = 0.1 tokens, well below 1

		if lim.Allow() {
			t.Fatalf("Allow() = true with ~0.1 tokens, want false; "+
				"fractional tokens below 1 must not pass threshold")
		}
	})
}

func TestTokens(t *testing.T) {
	t.Parallel()

	t.Run("happy_path_initial_equals_burst", func(t *testing.T) {
		t.Parallel()
		lim := New(0, 10)

		got := lim.Tokens()
		if got < 9.9 || got > 10.1 {
			t.Errorf("Tokens() = %f, want ≈ 10.0", got)
		}
	})

	t.Run("reflects_remaining_after_consumption", func(t *testing.T) {
		t.Parallel()
		lim := New(0, 5)
		lim.Allow()
		lim.Allow()

		got := lim.Tokens()
		if got < 2.9 || got > 3.1 {
			t.Errorf("Tokens() = %f after consuming 2 of 5, want ≈ 3.0", got)
		}
	})

	t.Run("zero_when_fully_exhausted", func(t *testing.T) {
		t.Parallel()
		lim := New(0, 2)
		lim.Allow()
		lim.Allow()

		got := lim.Tokens()
		if got < -0.1 || got > 0.1 {
			t.Errorf("Tokens() = %f after full exhaustion, want ≈ 0.0", got)
		}
	})

	t.Run("capped_at_burst_after_refill", func(t *testing.T) {
		t.Parallel()
		lim := New(1000, 5)
		time.Sleep(50 * time.Millisecond) // 1000*0.05 = 50 tokens, capped at 5

		got := lim.Tokens()
		if got > 5.1 {
			t.Errorf("Tokens() = %f, want <= 5.0 (burst cap)", got)
		}
		if got < 4.5 {
			t.Errorf("Tokens() = %f, want ≈ 5.0", got)
		}
	})

	t.Run("killer/read_only_no_state_mutation", func(t *testing.T) {
		t.Parallel()
		// H8: if Tokens() mutates l.lastTime or l.tokens,
		// subsequent Allow() calls see incorrect token counts.
		lim := New(0, 3)

		for range 100 {
			lim.Tokens()
		}

		allowed := 0
		for range 3 {
			if lim.Allow() {
				allowed++
			}
		}
		if allowed != 3 {
			t.Fatalf("allowed = %d after 100 Tokens() calls, want 3; "+
				"Tokens() must not mutate internal state", allowed)
		}
	})
}

func TestAllow_Concurrent(t *testing.T) {
	t.Parallel()

	t.Run("killer/no_over_consumption_under_contention", func(t *testing.T) {
		t.Parallel()
		// H6: without proper mutex, concurrent Allow() calls could
		// read-modify-write tokens non-atomically, allowing more than burst.
		const burst = 100
		const goroutines = 500
		lim := New(0, burst) // rate=0 for determinism

		var allowed int64
		var wg sync.WaitGroup

		for range goroutines {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if lim.Allow() {
					atomic.AddInt64(&allowed, 1)
				}
			}()
		}
		wg.Wait()

		if allowed != burst {
			t.Fatalf("concurrent Allow: allowed = %d, want exactly %d (no over-consumption)", allowed, burst)
		}
	})

	t.Run("concurrent_allow_and_tokens_no_race", func(t *testing.T) {
		t.Parallel()
		lim := New(100, 50)

		var wg sync.WaitGroup
		for range 100 {
			wg.Add(2)
			go func() {
				defer wg.Done()
				lim.Allow()
			}()
			go func() {
				defer wg.Done()
				lim.Tokens()
			}()
		}
		wg.Wait()
	})
}
