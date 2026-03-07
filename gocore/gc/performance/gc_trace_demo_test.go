package performance

import (
	"fmt"
	"runtime"
	"testing"
)

// printMemStats 打印关键的 GC 和内存指标
func printMemStats(label string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("[%s]\n", label)
	fmt.Printf("  HeapAlloc    = %d KiB\n", m.HeapAlloc/1024)
	fmt.Printf("  HeapObjects  = %d\n", m.HeapObjects)
	fmt.Printf("  NumGC        = %d\n", m.NumGC)
	fmt.Printf("  PauseTotalNs = %d μs\n", m.PauseTotalNs/1000)
	fmt.Printf("  LastPause    = %d μs\n", m.PauseNs[(m.NumGC+255)%256]/1000)
	fmt.Println()
}

// TestGCTraceDemo 演示如何通过 runtime.ReadMemStats 观测 GC 行为
// 运行方式: go test -v -run TestGCTraceDemo ./goprincipleandpractise/gc/performance/
// 配合 gctrace: GODEBUG=gctrace=1 go test -v -run TestGCTraceDemo ./goprincipleandpractise/gc/performance/
func TestGCTraceDemo(t *testing.T) {
	printMemStats("初始状态")

	// 制造大量堆分配
	hold := make([]*[1024]byte, 0, 10000)
	for i := 0; i < 10000; i++ {
		hold = append(hold, new([1024]byte))
	}
	printMemStats("分配 10000 个 1KiB 对象后")

	// 手动触发 GC
	runtime.GC()
	printMemStats("手动 GC 后（对象仍被引用）")

	// 释放引用
	hold = nil
	runtime.GC()
	printMemStats("释放引用并 GC 后")
}
