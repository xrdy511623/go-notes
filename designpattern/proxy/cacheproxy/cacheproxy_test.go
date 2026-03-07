package cacheproxy

import (
	"context"
	"sync"
	"testing"
	"time"

	"go-notes/designpattern/proxy/fetcher"
)

func TestProxy_MissCallsReal(t *testing.T) {
	real := &fetcher.SlowFetcher{Latency: 0}
	p := New(real, time.Minute)

	v, err := p.Fetch(context.Background(), "key1")
	if err != nil {
		t.Fatal(err)
	}
	if v != "data-for-key1" {
		t.Fatalf("got %q, want %q", v, "data-for-key1")
	}
	if real.Calls() != 1 {
		t.Fatalf("expected 1 call to real, got %d", real.Calls())
	}
}

func TestProxy_HitSkipsReal(t *testing.T) {
	real := &fetcher.SlowFetcher{Latency: 0}
	p := New(real, time.Minute)

	ctx := context.Background()
	p.Fetch(ctx, "key1")
	p.Fetch(ctx, "key1")
	p.Fetch(ctx, "key1")

	if real.Calls() != 1 {
		t.Fatalf("expected 1 call to real (2 cache hits), got %d", real.Calls())
	}
}

func TestProxy_DifferentKeys(t *testing.T) {
	real := &fetcher.SlowFetcher{Latency: 0}
	p := New(real, time.Minute)

	ctx := context.Background()
	p.Fetch(ctx, "a")
	p.Fetch(ctx, "b")
	p.Fetch(ctx, "a")

	if real.Calls() != 2 {
		t.Fatalf("expected 2 calls (a + b), got %d", real.Calls())
	}
	if p.Len() != 2 {
		t.Fatalf("expected 2 cache entries, got %d", p.Len())
	}
}

func TestProxy_TTLExpiration(t *testing.T) {
	real := &fetcher.SlowFetcher{Latency: 0}
	p := New(real, 20*time.Millisecond)

	ctx := context.Background()
	p.Fetch(ctx, "key1")

	time.Sleep(30 * time.Millisecond)

	p.Fetch(ctx, "key1")

	if real.Calls() != 2 {
		t.Fatalf("expected 2 calls after TTL expiry, got %d", real.Calls())
	}
}

func TestProxy_ContextCancellation(t *testing.T) {
	real := &fetcher.SlowFetcher{Latency: time.Second}
	p := New(real, time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := p.Fetch(ctx, "key1")
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestProxy_ErrorNotCached(t *testing.T) {
	real := &fetcher.SlowFetcher{Latency: time.Second}
	p := New(real, time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	p.Fetch(ctx, "key1")

	if p.Len() != 0 {
		t.Fatal("failed requests should not be cached")
	}
}

func TestProxy_ConcurrentAccess(t *testing.T) {
	real := &fetcher.SlowFetcher{Latency: 0}
	p := New(real, time.Minute)

	var wg sync.WaitGroup
	ctx := context.Background()

	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, err := p.Fetch(ctx, "shared")
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if v != "data-for-shared" {
				t.Errorf("got %q, want %q", v, "data-for-shared")
			}
		}()
	}
	wg.Wait()
}

func TestProxy_SatisfiesFetcherInterface(t *testing.T) {
	real := &fetcher.SlowFetcher{}
	var _ fetcher.Fetcher = New(real, time.Minute)
}
