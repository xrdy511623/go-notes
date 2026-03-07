package performance

import (
	"os"
	"runtime"
	"runtime/pprof"
	"testing"
)

// TestManualCPUProfile 演示 runtime/pprof 手动采集 CPU profile
// 适用于非 HTTP 服务场景（CLI 工具、批处理任务等）
//
// 运行:
//
//	go test -v -run TestManualCPUProfile ./goprincipleandpractise/pprof-practise/performance/
//
// 查看:
//
//	go tool pprof cpu_manual.prof
func TestManualCPUProfile(t *testing.T) {
	f, err := os.Create("cpu_manual.prof")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := pprof.StartCPUProfile(f); err != nil {
		t.Fatal(err)
	}
	defer pprof.StopCPUProfile()

	// 模拟 CPU 密集工作
	result := 0
	for i := 0; i < 10000000; i++ {
		result += i * i
	}
	_ = result

	t.Log("CPU profile 已写入 cpu_manual.prof")
	t.Log("查看方式: go tool pprof cpu_manual.prof")
}

// TestManualHeapProfile 演示 runtime/pprof 手动采集 heap profile
//
// 运行:
//
//	go test -v -run TestManualHeapProfile ./goprincipleandpractise/pprof-practise/performance/
//
// 查看:
//
//	go tool pprof -inuse_space mem_manual.prof
func TestManualHeapProfile(t *testing.T) {
	// 制造一些堆分配
	hold := make([][]byte, 0, 1000)
	for i := 0; i < 1000; i++ {
		hold = append(hold, make([]byte, 1024))
	}

	// 先触发 GC 确保数据准确
	runtime.GC()

	f, err := os.Create("mem_manual.prof")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := pprof.WriteHeapProfile(f); err != nil {
		t.Fatal(err)
	}

	t.Logf("Heap profile 已写入 mem_manual.prof (持有 %d 个对象)", len(hold))
	t.Log("查看方式: go tool pprof -inuse_space mem_manual.prof")
}
