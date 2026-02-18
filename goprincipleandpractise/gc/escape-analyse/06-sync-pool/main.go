package main

import (
	"fmt"
	"sync"
)

// sync.Pool 的 interface{} 逃逸演示
//
// sync.Pool 的 Put/Get 参数类型为 interface{}，放入 Pool 的对象必然逃逸到堆上。
// Pool 的价值不是消除逃逸，而是通过复用减少分配次数，从而降低 GC 压力。
//
// 逃逸分析:
//   go build -gcflags=-m ./goprincipleandpractise/gc/escape-analyse/06-sync-pool/
//
// 预期输出:
//   &bytes.Buffer{} escapes to heap  (因为赋值给 interface{})

type Buffer struct {
	data []byte
}

var pool = sync.Pool{
	New: func() interface{} {
		return &Buffer{data: make([]byte, 0, 1024)}
	},
}

func process() {
	// Get 返回 interface{}，需要类型断言
	buf := pool.Get().(*Buffer)

	// 使用 buffer
	buf.data = append(buf.data[:0], "hello, pool"...)
	fmt.Println(string(buf.data))

	// 归还到 Pool
	pool.Put(buf) // buf 逃逸：装箱为 interface{}
}

func main() {
	// 第一次调用：Pool 为空，调用 New 创建对象
	process()

	// 第二次调用：从 Pool 中复用对象，避免新分配
	process()
}
