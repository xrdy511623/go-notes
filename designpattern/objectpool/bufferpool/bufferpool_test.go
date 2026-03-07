package bufferpool

import (
	"bytes"
	"sync"
	"testing"
)

func TestGet_ReturnsEmptyBuffer(t *testing.T) {
	p := New(256)
	buf := p.Get()
	defer p.Put(buf)

	if buf.Len() != 0 {
		t.Fatalf("expected empty buffer, got len=%d", buf.Len())
	}
}

func TestGet_HasInitialCapacity(t *testing.T) {
	p := New(1024)
	buf := p.Get()
	defer p.Put(buf)

	if buf.Cap() < 1024 {
		t.Fatalf("expected cap >= 1024, got %d", buf.Cap())
	}
}

func TestGet_ResetsReusedBuffer(t *testing.T) {
	p := New(256)

	buf := p.Get()
	buf.WriteString("dirty data that should not persist")
	p.Put(buf)

	reused := p.Get()
	defer p.Put(reused)
	if reused.Len() != 0 {
		t.Fatalf("expected empty buffer after reuse, got len=%d data=%q", reused.Len(), reused.String())
	}
}

func TestPut_DiscardsOversized(t *testing.T) {
	p := New(256)

	huge := bytes.NewBuffer(make([]byte, 0, 128<<10)) // 128KB > 64KB limit
	p.Put(huge)

	buf := p.Get()
	defer p.Put(buf)
	if buf.Cap() > defaultMaxCap {
		t.Fatalf("expected normal capacity, got cap=%d (oversized was not discarded)", buf.Cap())
	}
}

func TestPut_Nil(t *testing.T) {
	p := New(256)
	p.Put(nil) // should not panic
}

func TestPool_ConcurrentAccess(t *testing.T) {
	p := New(256)

	var wg sync.WaitGroup
	const goroutines = 100
	const iterations = 1000

	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			for range iterations {
				buf := p.Get()
				buf.WriteString("concurrent write test data payload")
				if buf.Len() == 0 {
					t.Error("buffer should not be empty after write")
				}
				p.Put(buf)
			}
		}()
	}
	wg.Wait()
}

func TestGet_ReusesCapacity(t *testing.T) {
	p := New(256)

	buf := p.Get()
	buf.WriteString("grow the buffer a bit beyond initial capacity ")
	for range 10 {
		buf.WriteString("padding; ")
	}
	grownCap := buf.Cap()
	p.Put(buf)

	reused := p.Get()
	defer p.Put(reused)

	if reused.Cap() < grownCap {
		t.Logf("buffer was not reused (new from pool.New), cap=%d vs grown=%d", reused.Cap(), grownCap)
	}
}

// benchSink 防止编译器将 buffer 优化到栈上，确保基准测试反映真实堆分配。
var benchSink *bytes.Buffer

// BenchmarkWithPool 展示对象池的核心价值：减少堆分配。
// 对比 BenchmarkWithoutPool，allocs/op 应接近 0。
func BenchmarkWithPool(b *testing.B) {
	p := New(512)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf := p.Get()
		for range 10 {
			buf.WriteString("benchmark payload data; ")
		}
		benchSink = buf
		p.Put(buf)
	}
}

func BenchmarkWithoutPool(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf := bytes.NewBuffer(make([]byte, 0, 512))
		for range 10 {
			buf.WriteString("benchmark payload data; ")
		}
		benchSink = buf
	}
}
