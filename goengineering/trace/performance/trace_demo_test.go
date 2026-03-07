package performance

import (
	"context"
	"fmt"
	"os"
	"runtime/trace"
	"sync"
	"testing"
)

// TestTraceDemo 演示 go tool trace 的使用方式
//
// 运行方式:
//
//	go test -v -run TestTraceDemo ./goprincipleandpractise/trace/performance/
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

// TestTraceLog 演示 trace.Log 的使用方式
//
// trace.Log 用于在 trace 时间线上标记关键业务事件，这些事件会出现在
// "User-defined tasks" 视图中对应 Task 的详情里，帮助关联运行时行为与业务语义。
//
// 运行方式:
//
//	go test -v -run TestTraceLog ./goprincipleandpractise/trace/performance/
//
// 生成 trace_log.out 后查看:
//
//	go tool trace trace_log.out
//
// 在 "User-defined tasks" 视图中，点击 "handleRequest" task，
// 可以看到每个阶段的 Region 耗时以及 Log 标记的业务事件。
func TestTraceLog(t *testing.T) {
	f, err := os.Create("trace_log.out")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := trace.Start(f); err != nil {
		t.Fatal(err)
	}
	defer trace.Stop()

	// 模拟处理多个请求
	var wg sync.WaitGroup
	for reqID := 1; reqID <= 3; reqID++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			handleRequest(context.Background(), id)
		}(reqID)
	}
	wg.Wait()

	t.Log("trace 已写入 trace_log.out，请运行: go tool trace trace_log.out")
}

// handleRequest 模拟一个请求的处理流程，展示 trace.Log 的典型用法
func handleRequest(ctx context.Context, reqID int) {
	// 为每个请求创建独立的 Task，方便在 trace UI 中按请求筛选
	ctx, task := trace.NewTask(ctx, "handleRequest")
	defer task.End()

	// trace.Log：在 trace 时间线上记录业务事件
	// 第一个参数是 category（分类），第二个是 message
	// 这些日志会出现在 Task 详情中，帮助将运行时行为与业务语义关联起来
	trace.Log(ctx, "requestID", fmt.Sprintf("req-%d", reqID))

	// 阶段1：输入验证
	trace.WithRegion(ctx, "validateInput", func() {
		trace.Log(ctx, "validation", "starting input validation")
		sum := 0
		for i := 0; i < 100000; i++ {
			sum += i
		}
		_ = sum
		trace.Log(ctx, "validation", "input validation passed")
	})

	// 阶段2：模拟数据库查询
	trace.WithRegion(ctx, "queryDB", func() {
		trace.Log(ctx, "db", fmt.Sprintf("querying order for req-%d", reqID))
		sum := 0
		for i := 0; i < 500000; i++ {
			sum += i
		}
		_ = sum
		trace.Log(ctx, "db", "query completed, 42 rows returned")
	})

	// 阶段3：渲染响应
	trace.WithRegion(ctx, "renderResponse", func() {
		trace.Log(ctx, "render", "rendering JSON response")
		sum := 0
		for i := 0; i < 200000; i++ {
			sum += i
		}
		_ = sum
	})

	trace.Log(ctx, "status", fmt.Sprintf("req-%d completed successfully", reqID))
}
