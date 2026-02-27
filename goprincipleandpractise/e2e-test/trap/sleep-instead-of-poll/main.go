package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

// ❌ 反面教材：用 time.Sleep 等待服务就绪
//
// 问题：E2E 测试启动环境后，用固定的 time.Sleep 等待服务就绪。
// 固定等待时间不可靠——太短服务未就绪导致测试失败，太长浪费时间。
// 而且在不同机器上（CI vs 本地），服务启动时间差异巨大。
//
// 根因：用"猜测"代替"确认"，没有真正验证服务是否就绪。

// ========== 反面示范 ==========

func badWaitForService() {
	fmt.Println("starting docker-compose...")
	// 假装启动了 docker-compose

	// ❌ 固定等待 10 秒，祈祷服务已经启动
	fmt.Println("waiting 10 seconds for service to be ready...")
	time.Sleep(10 * time.Second)
	// 在快速机器上浪费 8 秒，在慢速机器上可能还不够

	fmt.Println("assuming service is ready, running tests...")
	// 如果服务还没就绪，测试会因为 connection refused 而失败
	// 开发者会觉得是"偶发问题"，于是把 Sleep 改成 20 秒...30 秒...
}

// ========== 正确做法 ==========

// ✅ 正确做法：轮询健康检查端点

func goodWaitForService(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			fmt.Printf("service ready after %v\n", timeout-time.Until(deadline))
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		// 短暂等待后重试
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("service not ready after %v", timeout)
}

// simulateServiceStartup 模拟一个随机时间后启动的服务
func simulateServiceStartup() {
	// 模拟服务启动需要 1-5 秒
	startupTime := time.Duration(1000+rand.Intn(4000)) * time.Millisecond

	go func() {
		time.Sleep(startupTime)
		mux := http.NewServeMux()
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		})
		http.ListenAndServe(":18080", mux)
	}()
}

func main() {
	fmt.Println("=== 反面教材：用 time.Sleep 等待服务就绪 ===")
	fmt.Println()
	fmt.Println("❌ 错误做法:")
	fmt.Println("  time.Sleep(10 * time.Second)")
	fmt.Println("  问题:")
	fmt.Println("  - 快速机器上白白浪费时间")
	fmt.Println("  - 慢速机器/CI 上时间不够")
	fmt.Println("  - 每次有人失败就加大 Sleep 时间 → 测试越来越慢")
	fmt.Println()
	fmt.Println("✅ 正确做法:")
	fmt.Println("  轮询健康检查端点，设置合理的超时上限")
	fmt.Println("  - 服务快速启动时立即开始测试")
	fmt.Println("  - 超时后明确报错（而非连接拒绝的混乱错误）")
	fmt.Println()

	// 演示轮询等待
	fmt.Println("--- 演示轮询等待 ---")
	simulateServiceStartup()
	err := goodWaitForService("http://localhost:18080/health", 10*time.Second)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Println("service is ready, tests can start!")
	}
}
