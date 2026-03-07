package buffersizetuning

import (
	"os"
	"testing"
)

/*
基准测试：不同 bufio.Reader 缓冲区大小对读取性能的影响

运行：
  go test -run '^$' -bench '^Benchmark' -benchtime=3s -count=3 -benchmem .

预期结论：
  - 512B → 4KB 提升显著（减少系统调用次数）
  - 4KB → 64KB 仍有改善
  - 64KB → 1MB 几乎无差别（OS 页缓存和预读机制已处理）
  - 最佳区间通常在 4KB-32KB
*/

const fileSize = 10 * 1024 * 1024 // 10MB

func benchmarkRead(b *testing.B, bufSize int) {
	b.Helper()
	f, err := CreateTestFile(fileSize)
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()

	b.ResetTimer()
	for b.Loop() {
		f.Seek(0, 0)
		if _, err := ReadWithBufferSize(f, bufSize); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadBuffer512B(b *testing.B)  { benchmarkRead(b, 512) }
func BenchmarkReadBuffer4KB(b *testing.B)   { benchmarkRead(b, 4*1024) }
func BenchmarkReadBuffer16KB(b *testing.B)  { benchmarkRead(b, 16*1024) }
func BenchmarkReadBuffer64KB(b *testing.B)  { benchmarkRead(b, 64*1024) }
func BenchmarkReadBuffer256KB(b *testing.B) { benchmarkRead(b, 256*1024) }
func BenchmarkReadBuffer1MB(b *testing.B)   { benchmarkRead(b, 1024*1024) }
