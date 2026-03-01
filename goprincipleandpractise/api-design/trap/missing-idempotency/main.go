// Package main 演示缺少幂等性保障的反模式。
//
// ❌ 错误: POST 创建接口无幂等性保障
//   - 网络超时重试 → 创建多个重复资源
//   - 客户端无法安全重试
//
// ✅ 正确: 使用 Idempotency-Key 头部
//   - 相同 Key 的重复请求返回缓存的第一次响应
//   - 客户端可以安全重试
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
)

func main() {
	fmt.Println("=== 缺少幂等性保障反模式 ===")
	fmt.Println()

	// ❌ 反模式: 无幂等性保障
	fmt.Println("❌ 反模式 — 无幂等性:")
	var badSeq int
	badHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		badSeq++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": badSeq, "name": "Alice"})
	})

	// 模拟网络超时导致的重试
	for i := 1; i <= 3; i++ {
		req := httptest.NewRequest("POST", "/api/v1/orders", strings.NewReader(`{"product":"book"}`))
		rec := httptest.NewRecorder()
		badHandler.ServeHTTP(rec, req)
		fmt.Printf("  第 %d 次请求: %s", i, rec.Body.String())
	}
	fmt.Println("  结果: 创建了 3 个重复订单!")

	// ✅ 正确模式: Idempotency-Key
	fmt.Println()
	fmt.Println("✅ 正确模式 — 使用 Idempotency-Key:")

	var (
		goodSeq int
		mu      sync.Mutex
		cache   = make(map[string][]byte)
	)

	goodHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("Idempotency-Key")
		w.Header().Set("Content-Type", "application/json")

		if key != "" {
			mu.Lock()
			if cached, ok := cache[key]; ok {
				mu.Unlock()
				w.Header().Set("X-Idempotent-Replayed", "true")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write(cached)
				return
			}
			mu.Unlock()
		}

		goodSeq++
		resp, _ := json.Marshal(map[string]any{"id": goodSeq, "name": "Alice"})

		if key != "" {
			mu.Lock()
			cache[key] = resp
			mu.Unlock()
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(resp)
		_, _ = w.Write([]byte("\n"))
	})

	for i := 1; i <= 3; i++ {
		req := httptest.NewRequest("POST", "/api/v1/orders", strings.NewReader(`{"product":"book"}`))
		req.Header.Set("Idempotency-Key", "order-abc-123")
		rec := httptest.NewRecorder()
		goodHandler.ServeHTTP(rec, req)
		replayed := rec.Header().Get("X-Idempotent-Replayed")
		fmt.Printf("  第 %d 次请求: %s  replayed=%s\n", i, strings.TrimSpace(rec.Body.String()), replayed)
	}
	fmt.Println("  结果: 只创建了 1 个订单，后续请求返回缓存响应。")
}
