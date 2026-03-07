package restful

import (
	"net/http"
	"time"
)

// NewServer 创建并配置 HTTP 服务器，演示 Go 1.22+ 路由语法。
//
// 路由设计要点:
//   - 使用 "METHOD /path" 格式（Go 1.22+）
//   - 资源名用复数名词 (/users, /orders)
//   - 路径参数用 {name} 占位符
//   - 版本号放在 URL 路径中 (/api/v1/...)
//
// 中间件链顺序:
//
//	Recovery → CORS → Trace → Logging → RateLimit → Auth → DeprecationHeaders → Handler
func NewServer() http.Handler {
	mux := http.NewServeMux()
	store := NewInMemoryUserStore()
	handler := NewUserHandler(store)
	limiter := NewRateLimiter(100, 60*time.Second) // 100 req/min

	sunset := time.Date(2026, time.December, 31, 0, 0, 0, 0, time.UTC)

	// v1 受保护路由（认证 + 授权 + 弃用执行机制）
	v1 := Chain(Recovery, CORS, Trace, Logging, limiter.Middleware, Auth(nil), DeprecationHeaders(sunset, "/api/v2/users"))

	// v2 受保护路由（不注入 v1 弃用头）
	v2 := Chain(Recovery, CORS, Trace, Logging, limiter.Middleware, Auth(nil))

	// ── v1 路由 ─────────────────────────────────────────
	mux.Handle("GET /api/v1/users",
		v1(http.HandlerFunc(handler.ListUsers)))
	mux.Handle("POST /api/v1/users",
		v1(http.HandlerFunc(handler.CreateUser)))
	mux.Handle("GET /api/v1/users/{id}",
		v1(http.HandlerFunc(handler.GetUser)))
	mux.Handle("PUT /api/v1/users/{id}",
		v1(http.HandlerFunc(handler.ReplaceUser)))
	mux.Handle("PATCH /api/v1/users/{id}",
		v1(http.HandlerFunc(handler.PatchUser)))
	mux.Handle("DELETE /api/v1/users/{id}",
		v1(http.HandlerFunc(handler.DeleteUser)))

	// ── v2 路由（示例：版本共存）───────────────────────────
	mux.Handle("GET /api/v2/users",
		v2(http.HandlerFunc(handler.ListUsers)))
	mux.Handle("GET /api/v2/users/{id}",
		v2(http.HandlerFunc(handler.GetUser)))

	// ── 健康检查 ────────────────────────────────────────
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	return mux
}
