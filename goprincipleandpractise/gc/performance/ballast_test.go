package performance

import (
	"fmt"
	"runtime"
	"testing"
)

// TestBallastEffect 演示 ballast 对 GC 频率的影响
// 运行: go test -v -run TestBallastEffect ./goprincipleandpractise/gc/performance/
func TestBallastEffect(t *testing.T) {
	allocateAndCountGC := func(label string, ballastSize int) {
		// 可选的 ballast
		var ballast []byte
		if ballastSize > 0 {
			ballast = make([]byte, ballastSize)
		}

		var before runtime.MemStats
		runtime.ReadMemStats(&before)
		startGC := before.NumGC

		// 模拟业务：循环分配临时对象
		for i := 0; i < 1000; i++ {
			tmp := make([]byte, 10*1024) // 10 KiB
			_ = tmp
		}

		var after runtime.MemStats
		runtime.ReadMemStats(&after)

		fmt.Printf("[%s] GC 次数: %d, HeapAlloc: %d KiB\n",
			label, after.NumGC-startGC, after.HeapAlloc/1024)

		_ = ballast // 防止 ballast 被优化掉
	}

	runtime.GC()
	allocateAndCountGC("无 ballast", 0)

	runtime.GC()
	allocateAndCountGC("100MB ballast", 100<<20)
}
