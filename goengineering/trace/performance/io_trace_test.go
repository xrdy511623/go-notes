package performance

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime/trace"
	"sync"
	"testing"
	"time"
)

// TestIOTrace 演示如何使用 trace 分析 I/O 阻塞场景
//
// 本测试启动一个本地 HTTP 服务器（模拟慢速下游服务），然后分别用串行和并发方式发起请求，
// 通过 trace 的 Network blocking profile 和 Syscall profile 对比两种方式的差异。
//
// 运行方式:
//
//	go test -v -run TestIOTrace ./goprincipleandpractise/trace/performance/
//
// 生成 io_trace.out 后查看:
//
//	go tool trace io_trace.out
//
// 重点关注:
//   - Network blocking profile：串行版本的网络等待时间是并发版本的数倍
//   - View trace by proc：串行版本只有一个 P 交替工作，并发版本多个 P 同时工作
//   - User-defined tasks/regions：各阶段的精确耗时
func TestIOTrace(t *testing.T) {
	// 启动模拟的慢速 HTTP 服务器
	server := startSlowServer(t)
	defer server.Close()

	baseURL := "http://" + server.Addr

	f, err := os.Create("io_trace.out")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := trace.Start(f); err != nil {
		t.Fatal(err)
	}
	defer trace.Stop()

	ctx, task := trace.NewTask(context.Background(), "IOTraceComparison")
	defer task.End()

	const numRequests = 5

	// 串行请求：逐个发起，总耗时 = 单次耗时 × numRequests
	trace.WithRegion(ctx, "serialRequests", func() {
		trace.Log(ctx, "phase", "starting serial requests")
		for i := 0; i < numRequests; i++ {
			doHTTPGet(ctx, baseURL, i)
		}
		trace.Log(ctx, "phase", "serial requests completed")
	})

	// 并发请求：同时发起，总耗时 ≈ 单次耗时
	trace.WithRegion(ctx, "concurrentRequests", func() {
		trace.Log(ctx, "phase", "starting concurrent requests")
		var wg sync.WaitGroup
		for i := 0; i < numRequests; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				doHTTPGet(ctx, baseURL, id)
			}(i)
		}
		wg.Wait()
		trace.Log(ctx, "phase", "concurrent requests completed")
	})

	t.Log("trace 已写入 io_trace.out，请运行: go tool trace io_trace.out")
}

// doHTTPGet 发起一次 HTTP GET 请求并用 Region 标记
func doHTTPGet(ctx context.Context, baseURL string, id int) {
	trace.WithRegion(ctx, fmt.Sprintf("httpGet-%d", id), func() {
		resp, err := http.Get(baseURL + "/slow")
		if err != nil {
			return
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	})
}

// startSlowServer 启动一个模拟慢速响应的本地 HTTP 服务器
func startSlowServer(t *testing.T) *http.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond) // 模拟 50ms 的下游延迟
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	server := &http.Server{
		Addr:    listener.Addr().String(),
		Handler: mux,
	}

	go server.Serve(listener)

	// 等待服务器就绪
	time.Sleep(10 * time.Millisecond)

	return server
}
