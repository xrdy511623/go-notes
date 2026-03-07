// Package main 演示动词 URL 反模式。
//
// ❌ 错误: 在 URL 中使用动词
//
//	POST /api/v1/createUser
//	POST /api/v1/deleteUser?id=123
//	GET  /api/v1/getUserById?id=123
//
// ✅ 正确: 使用 HTTP 方法 + 名词资源
//
//	POST   /api/v1/users          → 创建
//	DELETE /api/v1/users/{id}     → 删除
//	GET    /api/v1/users/{id}     → 查询
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
)

func main() {
	fmt.Println("=== 动词 URL 反模式 ===")
	fmt.Println()

	// ❌ 反模式: 动词 URL
	badMux := http.NewServeMux()
	badMux.HandleFunc("POST /api/v1/createUser", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "1", "name": "Alice"})
	})
	badMux.HandleFunc("GET /api/v1/getUserById", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"id": r.URL.Query().Get("id")})
	})
	badMux.HandleFunc("POST /api/v1/deleteUser", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	fmt.Println("❌ 反模式 — 动词 URL:")
	fmt.Println("  POST /api/v1/createUser")
	fmt.Println("  GET  /api/v1/getUserById?id=1")
	fmt.Println("  POST /api/v1/deleteUser?id=1")

	req := httptest.NewRequest("POST", "/api/v1/createUser", nil)
	rec := httptest.NewRecorder()
	badMux.ServeHTTP(rec, req)
	fmt.Printf("  响应: %d %s\n", rec.Code, rec.Body.String())

	// ✅ 正确模式: RESTful 资源 URL
	goodMux := http.NewServeMux()
	goodMux.HandleFunc("POST /api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "1", "name": "Alice"})
	})
	goodMux.HandleFunc("GET /api/v1/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"id": r.PathValue("id")})
	})
	goodMux.HandleFunc("DELETE /api/v1/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	fmt.Println()
	fmt.Println("✅ 正确模式 — RESTful 资源 URL:")
	fmt.Println("  POST   /api/v1/users")
	fmt.Println("  GET    /api/v1/users/{id}")
	fmt.Println("  DELETE /api/v1/users/{id}")

	req = httptest.NewRequest("POST", "/api/v1/users", nil)
	rec = httptest.NewRecorder()
	goodMux.ServeHTTP(rec, req)
	fmt.Printf("  响应: %d %s", rec.Code, rec.Body.String())

	fmt.Println()
	fmt.Println("要点: HTTP 方法已经表达了动作语义，URL 只需表达资源。")
}
