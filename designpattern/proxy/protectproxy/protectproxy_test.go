package protectproxy

import (
	"context"
	"errors"
	"testing"

	"go-notes/designpattern/proxy/fetcher"
)

func TestProxy_AllowedRole(t *testing.T) {
	real := &fetcher.SlowFetcher{Latency: 0}
	p := New(real, "admin", "editor")

	ctx := WithRole(context.Background(), "admin")
	v, err := p.Fetch(ctx, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if v != "data-for-secret" {
		t.Fatalf("got %q, want %q", v, "data-for-secret")
	}
	if real.Calls() != 1 {
		t.Fatalf("expected 1 call to real, got %d", real.Calls())
	}
}

func TestProxy_DeniedRole(t *testing.T) {
	real := &fetcher.SlowFetcher{Latency: 0}
	p := New(real, "admin")

	ctx := WithRole(context.Background(), "guest")
	_, err := p.Fetch(ctx, "secret")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
	if real.Calls() != 0 {
		t.Fatalf("real should not be called for denied role, got %d calls", real.Calls())
	}
}

func TestProxy_NoRole(t *testing.T) {
	real := &fetcher.SlowFetcher{Latency: 0}
	p := New(real, "admin")

	_, err := p.Fetch(context.Background(), "secret")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for missing role, got %v", err)
	}
	if real.Calls() != 0 {
		t.Fatalf("real should not be called without role, got %d calls", real.Calls())
	}
}

func TestProxy_MultipleAllowedRoles(t *testing.T) {
	real := &fetcher.SlowFetcher{Latency: 0}
	p := New(real, "admin", "editor", "viewer")

	for _, role := range []string{"admin", "editor", "viewer"} {
		ctx := WithRole(context.Background(), role)
		_, err := p.Fetch(ctx, "key")
		if err != nil {
			t.Fatalf("role %q should be allowed, got %v", role, err)
		}
	}
	if real.Calls() != 3 {
		t.Fatalf("expected 3 calls, got %d", real.Calls())
	}
}

func TestProxy_SatisfiesFetcherInterface(t *testing.T) {
	real := &fetcher.SlowFetcher{}
	var _ fetcher.Fetcher = New(real, "admin")
}
