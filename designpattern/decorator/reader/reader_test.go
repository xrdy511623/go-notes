package reader

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"
)

func TestCountingReader_BasicRead(t *testing.T) {
	data := "Hello, Decorator Pattern!"
	cr := NewCountingReader(strings.NewReader(data))

	got, err := io.ReadAll(cr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != data {
		t.Errorf("got %q, want %q", got, data)
	}
	if cr.BytesRead() != int64(len(data)) {
		t.Errorf("BytesRead() = %d, want %d", cr.BytesRead(), len(data))
	}
}

func TestCountingReader_EmptyReader(t *testing.T) {
	cr := NewCountingReader(strings.NewReader(""))
	_, _ = io.ReadAll(cr)
	if cr.BytesRead() != 0 {
		t.Errorf("BytesRead() = %d, want 0", cr.BytesRead())
	}
}

func TestCountingReader_ChunkedRead(t *testing.T) {
	data := "abcdefghij"
	cr := NewCountingReader(strings.NewReader(data))

	buf := make([]byte, 3)
	total := 0
	for {
		n, err := cr.Read(buf)
		total += n
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if int64(total) != cr.BytesRead() {
		t.Errorf("total=%d != BytesRead()=%d", total, cr.BytesRead())
	}
}

func TestCountingReader_ConcurrentBytesRead(t *testing.T) {
	data := bytes.Repeat([]byte("x"), 10000)
	cr := NewCountingReader(bytes.NewReader(data))

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		io.ReadAll(cr)
	}()
	wg.Wait()

	if cr.BytesRead() != int64(len(data)) {
		t.Errorf("BytesRead() = %d, want %d", cr.BytesRead(), len(data))
	}
}

func TestCountingReader_Composition(t *testing.T) {
	data := "layer upon layer"
	inner := NewCountingReader(strings.NewReader(data))
	outer := NewCountingReader(inner)

	_, _ = io.ReadAll(outer)

	if inner.BytesRead() != int64(len(data)) {
		t.Errorf("inner.BytesRead() = %d, want %d", inner.BytesRead(), len(data))
	}
	if outer.BytesRead() != int64(len(data)) {
		t.Errorf("outer.BytesRead() = %d, want %d", outer.BytesRead(), len(data))
	}
}

func TestProgressReader_ReportsProgress(t *testing.T) {
	data := "1234567890"
	total := int64(len(data))

	var reports []int64
	pr := NewProgressReader(strings.NewReader(data), total, func(bytesRead, _ int64) {
		reports = append(reports, bytesRead)
	})

	buf := make([]byte, 3)
	for {
		_, err := pr.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}

	if len(reports) == 0 {
		t.Fatal("expected progress reports")
	}
	last := reports[len(reports)-1]
	if last != total {
		t.Errorf("last report = %d, want %d", last, total)
	}
}

func TestProgressReader_NilCallback(t *testing.T) {
	data := "safe with nil"
	pr := NewProgressReader(strings.NewReader(data), int64(len(data)), nil)

	got, err := io.ReadAll(pr)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != data {
		t.Errorf("got %q, want %q", got, data)
	}
}

func TestProgressReader_BytesRead(t *testing.T) {
	data := "track bytes"
	pr := NewProgressReader(strings.NewReader(data), int64(len(data)), nil)

	_, _ = io.ReadAll(pr)
	if pr.BytesRead() != int64(len(data)) {
		t.Errorf("BytesRead() = %d, want %d", pr.BytesRead(), len(data))
	}
}

func TestComposition_CountingAndProgress(t *testing.T) {
	data := "composed decorators"
	total := int64(len(data))

	var lastProgress int64
	cr := NewCountingReader(
		NewProgressReader(strings.NewReader(data), total, func(bytesRead, _ int64) {
			lastProgress = bytesRead
		}),
	)

	_, _ = io.ReadAll(cr)

	if cr.BytesRead() != total {
		t.Errorf("BytesRead() = %d, want %d", cr.BytesRead(), total)
	}
	if lastProgress != total {
		t.Errorf("lastProgress = %d, want %d", lastProgress, total)
	}
}

func BenchmarkCountingReader(b *testing.B) {
	data := bytes.Repeat([]byte("benchmark"), 1000)
	buf := make([]byte, 4096)

	b.Run("plain_reader", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			r := bytes.NewReader(data)
			io.CopyBuffer(io.Discard, r, buf)
		}
	})

	b.Run("counting_reader", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			r := NewCountingReader(bytes.NewReader(data))
			io.CopyBuffer(io.Discard, r, buf)
		}
	})

	b.Run("double_wrapped", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			r := NewCountingReader(NewCountingReader(bytes.NewReader(data)))
			io.CopyBuffer(io.Discard, r, buf)
		}
	})
}
