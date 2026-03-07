package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
)

/*
陷阱：E2E 测试关注实现细节而非用户行为

运行：go run .

预期行为：
  启动一个模拟订单服务，分别演示"错误做法"和"正确做法"：
  - 错误做法：创建订单后，直接查内部存储验证（绕过 API）
  - 正确做法：创建订单后，通过 API 查询验证（用户视角）

  当内部实现变更（如存储字段名从 status 改为 state）时，
  错误做法的测试会挂，正确做法的测试不受影响。
*/

// orderStore 模拟内部存储（数据库）
type orderStore struct {
	mu     sync.RWMutex
	orders map[string]map[string]any // 内部字段名可能随时变化
}

var store = &orderStore{orders: make(map[string]map[string]any)}

func newOrderServer() *httptest.Server {
	mux := http.NewServeMux()

	// POST /api/orders — 创建订单
	mux.HandleFunc("/api/orders", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Item     string `json:"item"`
			Quantity int    `json:"quantity"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		orderID := fmt.Sprintf("ORD-%d", len(store.orders)+1)

		// 内部存储：字段名是 internal_status（不是 status）
		store.mu.Lock()
		store.orders[orderID] = map[string]any{
			"id":              orderID,
			"item":            req.Item,
			"quantity":        req.Quantity,
			"internal_status": "pending", // 注意：内部字段名！
		}
		store.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		// API 返回给用户的字段名是 status（对外接口）
		json.NewEncoder(w).Encode(map[string]any{
			"id":     orderID,
			"status": "pending",
		})
	})

	// GET /api/orders/{id} — 查询订单
	mux.HandleFunc("/api/orders/", func(w http.ResponseWriter, r *http.Request) {
		orderID := strings.TrimPrefix(r.URL.Path, "/api/orders/")
		store.mu.RLock()
		order, ok := store.orders[orderID]
		store.mu.RUnlock()
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		// API 对外暴露 status，而非内部的 internal_status
		json.NewEncoder(w).Encode(map[string]any{
			"id":     order["id"],
			"item":   order["item"],
			"status": order["internal_status"],
		})
	})

	return httptest.NewServer(mux)
}

func main() {
	server := newOrderServer()
	defer server.Close()

	fmt.Println("=== 模拟订单服务已启动 ===")
	fmt.Println()

	// Step 1: 创建订单
	resp, err := http.Post(server.URL+"/api/orders", "application/json",
		strings.NewReader(`{"item": "Go in Action", "quantity": 1}`))
	if err != nil {
		fmt.Printf("创建订单失败: %v\n", err)
		return
	}
	var created struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	json.NewDecoder(resp.Body).Decode(&created)
	resp.Body.Close()
	fmt.Printf("订单已创建: id=%s, status=%s\n\n", created.ID, created.Status)

	// ❌ 错误做法：直接查内部存储验证
	fmt.Println("--- ❌ 错误做法：直接查内部存储（模拟查数据库）---")
	store.mu.RLock()
	internal := store.orders[created.ID]
	store.mu.RUnlock()

	// 试图用内部字段名 "status" 查找——但内部用的是 "internal_status"
	internalStatus, ok := internal["status"]
	if !ok {
		fmt.Println("  查找字段 'status' → 不存在！")
		fmt.Println("  原因: 内部存储用的是 'internal_status'，不是 'status'")
		fmt.Println("  如果测试写 assert.Equal(t, \"pending\", order[\"status\"]) → 测试失败")
	} else {
		fmt.Printf("  内部 status = %v\n", internalStatus)
	}

	// 用正确的内部字段名能拿到
	fmt.Printf("  内部 internal_status = %v（内部字段名）\n", internal["internal_status"])
	fmt.Println()
	fmt.Println("  问题:")
	fmt.Println("  - 测试与内部实现强耦合")
	fmt.Println("  - 字段名重构后测试全挂，但 API 行为完全没变")
	fmt.Println("  - E2E 测试本不应该知道内部存储结构")

	fmt.Println()

	// ✅ 正确做法：通过 API 查询验证
	fmt.Println("--- ✅ 正确做法：通过 API 查询验证（用户视角）---")
	getResp, err := http.Get(server.URL + "/api/orders/" + created.ID)
	if err != nil {
		fmt.Printf("  查询失败: %v\n", err)
		return
	}
	var fetched struct {
		ID     string `json:"id"`
		Item   string `json:"item"`
		Status string `json:"status"`
	}
	json.NewDecoder(getResp.Body).Decode(&fetched)
	getResp.Body.Close()

	fmt.Printf("  GET /api/orders/%s → status=%s\n", created.ID, fetched.Status)
	if fetched.Status == "pending" {
		fmt.Println("  验证通过: status == \"pending\"")
	}
	fmt.Println()
	fmt.Println("  优点:")
	fmt.Println("  - 不依赖内部字段名，内部重构不影响测试")
	fmt.Println("  - 测试验证的是用户看到的行为")
	fmt.Println("  - 内部从 'status' 改名为 'internal_status'，测试照样通过")
}
