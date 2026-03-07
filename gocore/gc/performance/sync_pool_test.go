package performance

import (
	"bytes"
	"sync"
	"testing"
)

var bufPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// BenchmarkWithoutPool 直接分配 buffer
func BenchmarkWithoutPool(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := new(bytes.Buffer)
		buf.WriteString("hello, world")
		_ = buf.String()
	}
}

// BenchmarkWithPool 使用 sync.Pool 复用 buffer
func BenchmarkWithPool(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := bufPool.Get().(*bytes.Buffer)
		buf.WriteString("hello, world")
		_ = buf.String()
		buf.Reset()
		bufPool.Put(buf)
	}
}

/*
运行:
go test -bench=Pool -benchmem ./goprincipleandpractise/gc/performance/

预期结果:
BenchmarkWithoutPool 的 allocs/op 明显高于 BenchmarkWithPool，
说明 sync.Pool 有效减少了堆分配次数。
*/
