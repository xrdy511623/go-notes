package performance

import (
	"context"
	"os"
	"runtime/trace"
	"sync"
	"testing"
)

// TestTraceDemo 演示 go tool trace 的使用方式
//
// 运行方式:
//
//	go test -v -run TestTraceDemo ./goprincipleandpractise/pprof-practise/performance/
//
// 生成 trace.out 后查看:
//
//	go tool trace trace.out
//
// 在浏览器中可以看到:
//   - Goroutine 调度时间线
//   - GC 暂停事件
//   - 系统调用阻塞
//   - 自定义 Task 和 Region 标记
func TestTraceDemo(t *testing.T) {
	f, err := os.Create("trace.out")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := trace.Start(f); err != nil {
		t.Fatal(err)
	}
	defer trace.Stop()

	// 自定义 Task：跨 goroutine 追踪一个业务流程
	ctx, task := trace.NewTask(context.Background(), "processOrders")
	defer task.End()

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// 自定义 Region：标记一段关键代码
			trace.WithRegion(ctx, "computeOrder", func() {
				result := 0
				for j := 0; j < 1000000; j++ {
					result += j
				}
				_ = result
			})
		}(i)
	}
	wg.Wait()

	t.Log("trace 已写入 trace.out，请运行: go tool trace trace.out")
}
