package scannervsreadline

import (
	"testing"
)

/*
基准测试：bufio.Scanner vs bufio.Reader.ReadString vs bufio.Reader.ReadLine

运行：
  go test -run '^$' -bench '^Benchmark' -benchtime=3s -count=3 -benchmem .

预期结论：
  - 短行（80字符）：三种方式性能接近，Scanner 略慢（SplitFunc 间接调用）
  - ReadString 每行分配一个 string，ReadLine 返回 []byte 避免分配
  - 长行（10KB）：Scanner 需要预设大缓冲区，ReadString 自动处理
*/

func BenchmarkScannerShortLines(b *testing.B) {
	for b.Loop() {
		r := GenerateLines(10000, 80)
		if _, err := ReadWithScanner(r, 0); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadStringShortLines(b *testing.B) {
	for b.Loop() {
		r := GenerateLines(10000, 80)
		if _, err := ReadWithReadString(r); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadLineShortLines(b *testing.B) {
	for b.Loop() {
		r := GenerateLines(10000, 80)
		if _, err := ReadWithReadLine(r); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkScannerLongLines(b *testing.B) {
	for b.Loop() {
		r := GenerateLines(1000, 10240)
		if _, err := ReadWithScanner(r, 1024*1024); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadStringLongLines(b *testing.B) {
	for b.Loop() {
		r := GenerateLines(1000, 10240)
		if _, err := ReadWithReadString(r); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadLineLongLines(b *testing.B) {
	for b.Loop() {
		r := GenerateLines(1000, 10240)
		if _, err := ReadWithReadLine(r); err != nil {
			b.Fatal(err)
		}
	}
}
