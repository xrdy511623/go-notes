package bufferedvsunbuffered

import (
	"os"
	"testing"
)

/*
基准测试：bufio 缓冲写入 vs 直接写入

运行：
  go test -run '^$' -bench '^Benchmark' -benchtime=3s -count=3 -benchmem .

预期结论：
  - 单字节写入时，bufio.Writer 比直接 os.File.Write 快 10-100 倍
  - 4KB 块写入时，差距缩小（接近 bufio 默认缓冲区大小）
  - 单字节读取时，bufio.Reader 同样大幅优于直接 os.File.Read
*/

const totalBytes = 256 * 1024 // 256KB

func createTempFile(b *testing.B) *os.File {
	b.Helper()
	f, err := os.CreateTemp("", "bench-buf-*")
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		f.Close()
		os.Remove(f.Name())
	})
	return f
}

func BenchmarkWriteUnbuffered1B(b *testing.B) {
	for b.Loop() {
		f := createTempFile(b)
		if err := WriteUnbuffered(f, 1, totalBytes); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriteBuffered1B(b *testing.B) {
	for b.Loop() {
		f := createTempFile(b)
		if err := WriteBuffered(f, 1, totalBytes); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriteUnbuffered64B(b *testing.B) {
	for b.Loop() {
		f := createTempFile(b)
		if err := WriteUnbuffered(f, 64, totalBytes); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriteBuffered64B(b *testing.B) {
	for b.Loop() {
		f := createTempFile(b)
		if err := WriteBuffered(f, 64, totalBytes); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriteUnbuffered4KB(b *testing.B) {
	for b.Loop() {
		f := createTempFile(b)
		if err := WriteUnbuffered(f, 4096, totalBytes); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWriteBuffered4KB(b *testing.B) {
	for b.Loop() {
		f := createTempFile(b)
		if err := WriteBuffered(f, 4096, totalBytes); err != nil {
			b.Fatal(err)
		}
	}
}

// prepareReadFile creates a temp file with totalBytes of data for read benchmarks.
func prepareReadFile(b *testing.B, size int) *os.File {
	b.Helper()
	f, err := os.CreateTemp("", "bench-read-*")
	if err != nil {
		b.Fatal(err)
	}
	data := make([]byte, size)
	for i := range data {
		data[i] = 'X'
	}
	f.Write(data)
	f.Seek(0, 0)
	b.Cleanup(func() {
		f.Close()
		os.Remove(f.Name())
	})
	return f
}

const readSize = 64 * 1024 // 64KB for read benchmarks

func BenchmarkReadUnbuffered1B(b *testing.B) {
	for b.Loop() {
		f := prepareReadFile(b, readSize)
		if _, err := ReadUnbuffered(f); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadBuffered1B(b *testing.B) {
	for b.Loop() {
		f := prepareReadFile(b, readSize)
		if _, err := ReadBuffered(f); err != nil {
			b.Fatal(err)
		}
	}
}
