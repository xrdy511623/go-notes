// Package unboundedspawn 演示不加限制地为每个请求启动 goroutine 的反模式。
//
// 问题：
//   - 没有并发上限 → 高负载下 goroutine 数量失控，最终 OOM
//   - 一个慢下游服务导致所有 goroutine 阻塞，连带拖垮其他健康服务的请求处理
//   - 不同下游之间没有隔离：Service A 变慢 → goroutine 积压 → 无资源处理 Service B 的请求
//   - 没有背压机制：上游无法感知下游已过载，继续全速发送请求
//
// 正确做法参考 semaphore/ 包（信号量舱壁）或 pool/ 包（工作池舱壁）。
package unboundedspawn

import (
	"fmt"
	"net/http"
	"time"
)

// Request 模拟一个需要处理的请求。
type Request struct {
	ID      string
	Payload string
}

// BadHandler 为每个请求无限制地启动 goroutine —— 这是反模式。
// 如果 processRequest 耗时较长（如调用慢 API），goroutine 会持续积压，
// 直到内存耗尽或 fd 耗尽。
type BadHandler struct{}

func (h *BadHandler) Handle(req Request) {
	go processRequest(req)
}

func processRequest(req Request) {
	time.Sleep(2 * time.Second)
	fmt.Printf("processed: %s\n", req.ID)
}

// BadHTTPHandler 是 HTTP 版本的反模式：在 handler 内部再次 spawn goroutine，
// 而 net/http 已经为每个连接启动了 goroutine，形成双层无限制派生。
func BadHTTPHandler(w http.ResponseWriter, r *http.Request) {
	go func() {
		time.Sleep(5 * time.Second)
		fmt.Println("background work done for", r.URL.Path)
	}()
	w.WriteHeader(http.StatusAccepted)
}
