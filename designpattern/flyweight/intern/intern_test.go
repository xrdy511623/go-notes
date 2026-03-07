package intern

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
	"unsafe"
)

func TestGet_ReturnsEqualString(t *testing.T) {
	p := New()
	got := p.Get("hello")
	if got != "hello" {
		t.Fatalf("expected %q, got %q", "hello", got)
	}
}

func TestGet_SharedPointer(t *testing.T) {
	p := New()
	s1 := string([]byte("hello"))
	s2 := string([]byte("hello"))

	r1 := p.Get(s1)
	r2 := p.Get(s2)

	if r1 != r2 {
		t.Fatal("interned strings should be equal")
	}
	if unsafe.StringData(r1) != unsafe.StringData(r2) {
		t.Fatal("interned strings should share the same backing array")
	}
}

func TestGet_MultipleDifferent(t *testing.T) {
	p := New()
	a := p.Get("alpha")
	b := p.Get("beta")
	if a == b {
		t.Fatal("different strings should not be equal")
	}
	if p.Len() != 2 {
		t.Fatalf("expected 2 entries, got %d", p.Len())
	}
}

func TestGet_EmptyString(t *testing.T) {
	p := New()
	got := p.Get("")
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
	if p.Len() != 1 {
		t.Fatal("empty string should be interned like any other string")
	}
}

func TestGet_SubstringIsolation(t *testing.T) {
	p := New()
	big := string(make([]byte, 4096))
	sub := big[100:105]

	interned := p.Get(sub)
	if len(interned) != 5 {
		t.Fatalf("expected len 5, got %d", len(interned))
	}
	// strings.Clone should produce an independent allocation,
	// not holding a reference to the 4KB buffer.
	if unsafe.StringData(interned) == unsafe.StringData(big) {
		t.Fatal("interned substring should be cloned away from the original buffer")
	}
}

func TestLen_Empty(t *testing.T) {
	p := New()
	if p.Len() != 0 {
		t.Fatalf("expected 0, got %d", p.Len())
	}
}

func TestLen_AfterInserts(t *testing.T) {
	p := New()
	p.Get("a")
	p.Get("b")
	p.Get("c")
	if p.Len() != 3 {
		t.Fatalf("expected 3, got %d", p.Len())
	}
}

func TestLen_DuplicatesNotCounted(t *testing.T) {
	p := New()
	p.Get("dup")
	p.Get("dup")
	p.Get("dup")
	if p.Len() != 1 {
		t.Fatalf("expected 1, got %d", p.Len())
	}
}

func TestHits_InitiallyZero(t *testing.T) {
	p := New()
	if p.Hits() != 0 {
		t.Fatalf("expected 0, got %d", p.Hits())
	}
}

func TestMisses_InitiallyZero(t *testing.T) {
	p := New()
	if p.Misses() != 0 {
		t.Fatalf("expected 0, got %d", p.Misses())
	}
}

func TestHitsMisses_Tracking(t *testing.T) {
	p := New()
	p.Get("x") // miss
	p.Get("y") // miss
	p.Get("x") // hit
	p.Get("x") // hit
	p.Get("y") // hit

	if p.Misses() != 2 {
		t.Fatalf("expected 2 misses, got %d", p.Misses())
	}
	if p.Hits() != 3 {
		t.Fatalf("expected 3 hits, got %d", p.Hits())
	}
}

func TestEvict_Existing(t *testing.T) {
	p := New()
	p.Get("gone")
	if !p.Evict("gone") {
		t.Fatal("expected Evict to return true for existing entry")
	}
	if p.Len() != 0 {
		t.Fatalf("expected 0 after evict, got %d", p.Len())
	}
}

func TestEvict_NonExistent(t *testing.T) {
	p := New()
	if p.Evict("nope") {
		t.Fatal("expected Evict to return false for non-existent entry")
	}
}

func TestEvict_UpdatesLen(t *testing.T) {
	p := New()
	p.Get("a")
	p.Get("b")
	p.Get("c")
	p.Evict("b")
	if p.Len() != 2 {
		t.Fatalf("expected 2 after evict, got %d", p.Len())
	}
}

func TestReset_ClearsAll(t *testing.T) {
	p := New()
	p.Get("a")
	p.Get("b")
	p.Reset()
	if p.Len() != 0 {
		t.Fatalf("expected 0 after reset, got %d", p.Len())
	}
}

func TestReset_ResetsCounters(t *testing.T) {
	p := New()
	p.Get("x")
	p.Get("x")
	p.Reset()
	if p.Hits() != 0 || p.Misses() != 0 {
		t.Fatalf("expected hits=0 misses=0 after reset, got hits=%d misses=%d",
			p.Hits(), p.Misses())
	}
}

func TestConcurrent_Get(t *testing.T) {
	p := New()
	words := []string{"error", "warning", "info", "debug", "trace"}
	const goroutines = 100
	const iterations = 200

	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range iterations {
				p.Get(words[i%len(words)])
			}
		}()
	}
	wg.Wait()

	if p.Len() != len(words) {
		t.Fatalf("expected %d unique entries, got %d", len(words), p.Len())
	}
	total := p.Hits() + p.Misses()
	expected := int64(goroutines * iterations)
	if total != expected {
		t.Fatalf("expected %d total lookups, got %d", expected, total)
	}
}

func TestConcurrent_GetAndEvict(t *testing.T) {
	p := New()
	const n = 100

	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", id%10)
			p.Get(key)
			if id%3 == 0 {
				p.Evict(key)
			}
		}(i)
	}
	wg.Wait()

	if p.Len() > 10 {
		t.Fatalf("expected at most 10 entries, got %d", p.Len())
	}
}

// --- Benchmarks ---

func BenchmarkGet_Hit(b *testing.B) {
	p := New()
	p.Get("hot-key")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Get("hot-key")
	}
}

func BenchmarkGet_Miss(b *testing.B) {
	p := New()
	keys := make([]string, b.N)
	for i := range keys {
		keys[i] = strconv.Itoa(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Get(keys[i])
	}
}

func BenchmarkGet_Parallel(b *testing.B) {
	p := New()
	words := []string{"error", "warning", "info", "debug", "trace"}
	for _, w := range words {
		p.Get(w)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			p.Get(words[i%len(words)])
			i++
		}
	})
}

func BenchmarkInternedVsRaw(b *testing.B) {
	words := []string{"error", "warning", "info", "debug", "trace"}

	b.Run("raw_alloc", func(b *testing.B) {
		var sink string
		for i := 0; i < b.N; i++ {
			sink = string([]byte(words[i%len(words)]))
		}
		_ = sink
	})

	b.Run("interned", func(b *testing.B) {
		p := New()
		var sink string
		for i := 0; i < b.N; i++ {
			sink = p.Get(string([]byte(words[i%len(words)])))
		}
		_ = sink
	})
}
