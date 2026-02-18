package performance

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"testing"
)

// TestGOMEMLIMITEffect 演示 GOMEMLIMIT 对 GC 行为的影响
// 运行: go test -v -run TestGOMEMLIMITEffect ./goprincipleandpractise/gc/performance/
func TestGOMEMLIMITEffect(t *testing.T) {
	allocateAndMeasure := func(label string) {
		var before runtime.MemStats
		runtime.ReadMemStats(&before)
		startGC := before.NumGC

		// 持续分配，保持部分对象存活
		hold := make([][]byte, 0, 500)
		for i := 0; i < 1000; i++ {
			b := make([]byte, 10*1024) // 10 KiB
			if i%2 == 0 {
				hold = append(hold, b) // 50% 存活
			}
		}

		var after runtime.MemStats
		runtime.ReadMemStats(&after)
		fmt.Printf("[%s] GC 次数: %d, HeapAlloc: %d KiB\n",
			label, after.NumGC-startGC, after.HeapAlloc/1024)
		_ = hold
	}

	// 场景1: 默认 GOGC=100
	runtime.GC()
	debug.SetGCPercent(100)
	allocateAndMeasure("GOGC=100, 无 GOMEMLIMIT")

	// 场景2: GOGC=off + GOMEMLIMIT
	runtime.GC()
	debug.SetGCPercent(-1)                 // 关闭按比例触发
	debug.SetMemoryLimit(50 * 1024 * 1024) // 50 MiB 软限制
	allocateAndMeasure("GOGC=off, GOMEMLIMIT=50MiB")

	// 恢复默认
	debug.SetGCPercent(100)
	debug.SetMemoryLimit(-1)
}
