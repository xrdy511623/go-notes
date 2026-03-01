package restful

import (
	"net/http"
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
//	Recovery → CORS → Logging → RateLimit → Auth → Handler
func NewServer() http.Handler {
	mux := http.NewServeMux()
	store := NewInMemoryUserStore()
	handler := NewUserHandler(store)
	limiter := NewRateLimiter(100, 60_000_000_000) // 100 req/min

	// 公开路由（不需要认证）
	public := Chain(Recovery, CORS, Logging, limiter.Middleware)

	// 受保护路由（需要认证）
	protected := Chain(Recovery, CORS, Logging, limiter.Middleware, Auth(nil))

	// ── v1 路由 ─────────────────────────────────────────
	// GET    /api/v1/users       → 列表
	// POST   /api/v1/users       → 创建
	// GET    /api/v1/users/{id}  → 详情
	// PUT    /api/v1/users/{id}  → 更新
	// DELETE /api/v1/users/{id}  → 删除

	mux.Handle("GET /api/v1/users",
		public(http.HandlerFunc(handler.ListUsers)))
	mux.Handle("POST /api/v1/users",
		protected(http.HandlerFunc(handler.CreateUser)))
	mux.Handle("GET /api/v1/users/{id}",
		public(http.HandlerFunc(handler.GetUser)))
	mux.Handle("PUT /api/v1/users/{id}",
		protected(http.HandlerFunc(handler.UpdateUser)))
	mux.Handle("DELETE /api/v1/users/{id}",
		protected(http.HandlerFunc(handler.DeleteUser)))

	// ── v2 路由（示例：版本共存）───────────────────────────
	// v2 可能返回不同的响应格式或增加字段
	mux.Handle("GET /api/v2/users",
		public(http.HandlerFunc(handler.ListUsers)))
	mux.Handle("GET /api/v2/users/{id}",
		public(http.HandlerFunc(handler.GetUser)))

	// ── 健康检查 ────────────────────────────────────────
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	return mux
}
