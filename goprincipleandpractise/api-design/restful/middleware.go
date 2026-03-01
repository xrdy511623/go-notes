package restful

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Middleware 定义中间件函数签名。
type Middleware func(http.Handler) http.Handler

// Chain 将多个中间件按顺序组合，执行顺序从左到右。
// 例如 Chain(Recovery, Logging, Auth) 的执行顺序:
//
//	Recovery → Logging → Auth → Handler
func Chain(middlewares ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}

// Recovery 捕获 handler 中的 panic，返回 500 而不是让服务器崩溃。
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[PANIC] %s %s: %v", r.Method, r.URL.Path, rec)
				WriteError(w, ErrServerFailure)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// responseRecorder 包装 ResponseWriter 以捕获状态码。
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}

// Logging 记录每个请求的方法、路径、状态码和耗时。
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf("[HTTP] %s %s → %d (%s)",
			r.Method, r.URL.Path, rec.statusCode, time.Since(start))
	})
}

// Auth 验证 Authorization 头部的 Bearer token。
// tokenValidator 用于自定义 token 验证逻辑；为 nil 时使用内置的 demo token。
func Auth(tokenValidator func(token string) bool) Middleware {
	if tokenValidator == nil {
		tokenValidator = func(token string) bool {
			return token == "demo-token" // 仅用于示例
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				WriteError(w, NewAppError(ErrUnauthorized, "missing or invalid Authorization header", nil))
				return
			}
			token := strings.TrimPrefix(auth, "Bearer ")
			if !tokenValidator(token) {
				WriteError(w, NewAppError(ErrUnauthorized, "invalid token", nil))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimiter 实现简单的滑动窗口限流器（基于固定窗口近似）。
// 生产环境建议使用 golang.org/x/time/rate 或分布式方案。
type RateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

// NewRateLimiter 创建限流器，limit 为窗口内最大请求数。
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

// Middleware 返回限流中间件。
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := r.RemoteAddr

		rl.mu.Lock()
		now := time.Now()
		windowStart := now.Add(-rl.window)

		// 清理过期记录
		reqs := rl.requests[clientIP]
		valid := reqs[:0]
		for _, t := range reqs {
			if t.After(windowStart) {
				valid = append(valid, t)
			}
		}

		if len(valid) >= rl.limit {
			rl.requests[clientIP] = valid
			rl.mu.Unlock()
			w.Header().Set("Retry-After", "60")
			WriteError(w, NewAppError(ErrRateLimited, "rate limit exceeded", nil))
			return
		}

		rl.requests[clientIP] = append(valid, now)
		rl.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}

// CORS 添加跨域资源共享头部。
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Idempotency-Key")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
