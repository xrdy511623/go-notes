package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"
)

/*
陷阱：E2E 测试之间没有数据隔离

运行：go run .

预期行为：
  启动一个用户注册服务，模拟两个测试场景：
  1. 使用固定邮箱注册 → 第二次运行必然冲突
  2. 使用唯一邮箱注册 → 永远不会冲突

  直接展示"第二次运行测试全挂"的问题。

  正确做法：每个测试使用唯一数据（时间戳/UUID 后缀）。
*/

type userDB struct {
	mu    sync.RWMutex
	users map[string]string // email → name
}

var db = &userDB{users: make(map[string]string)}

func newUserServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		db.mu.Lock()
		defer db.mu.Unlock()

		if _, exists := db.users[req.Email]; exists {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{
				"error": fmt.Sprintf("email %s already registered", req.Email),
			})
			return
		}
		db.users[req.Email] = req.Name
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "created"})
	})
	return httptest.NewServer(mux)
}

func register(serverURL, email, name string) (int, string) {
	body := fmt.Sprintf(`{"email":"%s","name":"%s"}`, email, name)
	resp, err := http.Post(serverURL+"/api/register", "application/json",
		strings.NewReader(body))
	if err != nil {
		return 0, err.Error()
	}
	defer resp.Body.Close()
	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if msg, ok := result["error"]; ok {
		return resp.StatusCode, msg
	}
	return resp.StatusCode, result["status"]
}

func main() {
	server := newUserServer()
	defer server.Close()

	fmt.Println("=== 模拟用户注册服务已启动 ===")
	fmt.Println()

	// ❌ 错误做法：使用固定邮箱
	fmt.Println("--- ❌ 错误做法：两个测试使用固定邮箱 ---")
	fmt.Println()

	fmt.Println("  [第一轮运行]")
	code1, msg1 := register(server.URL, "alice@test.com", "Alice")
	fmt.Printf("    TestRegisterA: POST /api/register (alice@test.com) → %d %s\n", code1, msg1)
	code2, msg2 := register(server.URL, "alice@test.com", "Alice2")
	fmt.Printf("    TestRegisterB: POST /api/register (alice@test.com) → %d %s\n", code2, msg2)
	fmt.Println()
	fmt.Println("  问题: TestRegisterB 返回 409，因为 TestRegisterA 已经注册了这个邮箱")
	fmt.Println("  单独运行 TestRegisterB 会通过，一起运行就失败 → Flaky!")

	fmt.Println()

	fmt.Println("  [模拟第二轮运行（不清理数据）]")
	code3, msg3 := register(server.URL, "alice@test.com", "Alice")
	fmt.Printf("    TestRegisterA: POST /api/register (alice@test.com) → %d %s\n", code3, msg3)
	fmt.Println("  问题: 连 TestRegisterA 都挂了，因为上一轮的数据还在")
	fmt.Println("  这就是为什么 CI 上有人说「重新跑一次就好了」")

	fmt.Println()
	fmt.Println()

	// ✅ 正确做法：每个测试使用唯一邮箱
	fmt.Println("--- ✅ 正确做法：每个测试使用唯一邮箱 ---")
	fmt.Println()

	for round := 1; round <= 3; round++ {
		fmt.Printf("  [第 %d 轮运行]\n", round)
		emailA := fmt.Sprintf("alice_%d_%d@test.com", round, time.Now().UnixNano())
		emailB := fmt.Sprintf("bob_%d_%d@test.com", round, time.Now().UnixNano())

		codeA, msgA := register(server.URL, emailA, "Alice")
		fmt.Printf("    TestRegisterA: %s → %d %s\n", emailA, codeA, msgA)
		codeB, msgB := register(server.URL, emailB, "Bob")
		fmt.Printf("    TestRegisterB: %s → %d %s\n", emailB, codeB, msgB)

		if codeA == 201 && codeB == 201 {
			fmt.Println("    全部通过")
		}
	}

	fmt.Println()
	fmt.Println("  无论运行多少轮，每个测试都使用唯一数据，永远不会冲突")
}
