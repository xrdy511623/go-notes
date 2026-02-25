package iocopyvsmanualloop

import (
	"bytes"
	"io"
	"testing"
)

/*
基准测试：io.Copy vs 手写循环 vs io.CopyBuffer

运行：
  go test -run '^$' -bench '^Benchmark' -benchtime=3s -count=3 -benchmem .

预期结论：
  - io.Copy + WriterTo 源（如 bytes.Reader）最快，跳过中间缓冲区
  - io.Copy 去掉 WriterTo 后与手写循环性能相当
  - io.CopyBuffer(4KB) 略慢于 32KB 默认缓冲区（更多迭代次数）
*/

const dataSize = 1024 * 1024 // 1MB

func generateData() []byte {
	data := make([]byte, dataSize)
	for i := range data {
		data[i] = byte(i % 256)
	}
	return data
}

func BenchmarkIOCopyWithWriterTo(b *testing.B) {
	data := generateData()
	b.ResetTimer()
	for b.Loop() {
		src := bytes.NewReader(data) // implements WriterTo
		dst := &bytes.Buffer{}
		dst.Grow(dataSize)
		CopyWithIOCopy(dst, src)
	}
}

func BenchmarkIOCopyWithoutWriterTo(b *testing.B) {
	data := generateData()
	b.ResetTimer()
	for b.Loop() {
		src := StripWriterTo(bytes.NewReader(data)) // no WriterTo
		dst := &bytes.Buffer{}
		dst.Grow(dataSize)
		CopyWithIOCopy(dst, src)
	}
}

func BenchmarkManualLoop32KB(b *testing.B) {
	data := generateData()
	b.ResetTimer()
	for b.Loop() {
		src := StripWriterTo(bytes.NewReader(data))
		dst := &bytes.Buffer{}
		dst.Grow(dataSize)
		CopyManualLoop(dst, src, 32*1024)
	}
}

func BenchmarkIOCopyBuffer4KB(b *testing.B) {
	data := generateData()
	b.ResetTimer()
	for b.Loop() {
		src := StripWriterTo(bytes.NewReader(data))
		dst := &bytes.Buffer{}
		dst.Grow(dataSize)
		CopyWithIOCopyBuffer(dst, src, 4*1024)
	}
}

func BenchmarkIOCopyBuffer32KB(b *testing.B) {
	data := generateData()
	b.ResetTimer()
	for b.Loop() {
		src := StripWriterTo(bytes.NewReader(data))
		dst := &bytes.Buffer{}
		dst.Grow(dataSize)
		CopyWithIOCopyBuffer(dst, src, 32*1024)
	}
}

func BenchmarkReadAllThenWrite(b *testing.B) {
	data := generateData()
	b.ResetTimer()
	for b.Loop() {
		src := bytes.NewReader(data)
		all, _ := io.ReadAll(src)
		dst := &bytes.Buffer{}
		dst.Grow(dataSize)
		dst.Write(all)
	}
}
