package restful

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Middleware 定义中间件函数签名。
type Middleware func(http.Handler) http.Handler

// Principal 表示认证后的调用方身份。
type Principal struct {
	Subject  string
	TenantID string
	Role     string
}

const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

type contextKey string

const (
	principalContextKey contextKey = "principal"
	traceContextKey     contextKey = "trace_id"
)

// PrincipalFromContext 从上下文提取认证身份。
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalContextKey).(Principal)
	return p, ok
}

// TraceIDFromContext 从上下文提取 trace_id。
func TraceIDFromContext(ctx context.Context) string {
	traceID, _ := ctx.Value(traceContextKey).(string)
	return traceID
}

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
				log.Printf("[PANIC] trace_id=%s method=%s path=%s panic=%v", TraceIDFromContext(r.Context()), r.Method, r.URL.Path, rec)
				WriteError(w, ErrServerFailure)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Trace 注入 trace_id/request_id，并通过响应头暴露。
func Trace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if traceID == "" {
			traceID = newTraceID()
		}

		ctx := context.WithValue(r.Context(), traceContextKey, traceID)
		w.Header().Set("X-Trace-ID", traceID)
		w.Header().Set("X-Request-ID", traceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func newTraceID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return strconvTimeFallback()
	}
	return hex.EncodeToString(buf)
}

func strconvTimeFallback() string {
	return "trace-" + time.Now().UTC().Format("20060102150405.000000000")
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

		principal, _ := PrincipalFromContext(r.Context())
		log.Printf("[HTTP] trace_id=%s method=%s path=%s status=%d duration=%s metric=http_requests_total tenant=%s subject=%s role=%s",
			TraceIDFromContext(r.Context()), r.Method, r.URL.Path, rec.statusCode, time.Since(start), principal.TenantID, principal.Subject, principal.Role)
	})
}

// TokenValidator 验证 token 并返回身份。
type TokenValidator func(token string) (Principal, bool)

// Auth 验证 Authorization 头部的 Bearer token，并把 Principal 写入上下文。
// tokenValidator 为 nil 时使用内置 demo token 解析逻辑。
func Auth(tokenValidator TokenValidator) Middleware {
	if tokenValidator == nil {
		tokenValidator = defaultTokenValidator
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				WriteError(w, NewAppError(ErrUnauthorized, "missing or invalid Authorization header", nil))
				return
			}

			token := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
			principal, ok := tokenValidator(token)
			if !ok {
				WriteError(w, NewAppError(ErrUnauthorized, "invalid token", nil))
				return
			}

			tenantHeader := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
			if tenantHeader != "" && tenantHeader != principal.TenantID {
				WriteError(w, ErrAccessDenied.WithDetail("tenant header does not match token tenant"))
				return
			}

			w.Header().Set("X-Audit-Subject", principal.Subject)
			w.Header().Set("X-Audit-Tenant", principal.TenantID)
			w.Header().Set("X-Audit-Role", principal.Role)
			ctx := context.WithValue(r.Context(), principalContextKey, principal)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func defaultTokenValidator(token string) (Principal, bool) {
	switch token {
	case "demo-token":
		return Principal{Subject: "demo-user", TenantID: "demo-tenant", Role: RoleUser}, true
	case "admin-token":
		return Principal{Subject: "demo-admin", TenantID: "demo-tenant", Role: RoleAdmin}, true
	}

	parts := strings.Split(token, ":")
	if len(parts) != 4 || parts[0] != "tenant" {
		return Principal{}, false
	}

	tenant := strings.TrimSpace(parts[1])
	role := strings.TrimSpace(parts[2])
	subject := strings.TrimSpace(parts[3])
	if tenant == "" || subject == "" {
		return Principal{}, false
	}
	if role != RoleUser && role != RoleAdmin {
		return Principal{}, false
	}

	return Principal{Subject: subject, TenantID: tenant, Role: role}, true
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
		clientIP := clientIPFromRemoteAddr(r.RemoteAddr)

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

func clientIPFromRemoteAddr(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil || host == "" {
		return remoteAddr
	}
	return host
}

// DeprecationHeaders 为 v1 端点注入弃用响应头。
func DeprecationHeaders(sunset time.Time, successorPath string) Middleware {
	sunsetValue := sunset.UTC().Format(http.TimeFormat)
	linkValue := "<" + successorPath + ">; rel=\"successor-version\""

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Deprecation", "true")
			w.Header().Set("Sunset", sunsetValue)
			w.Header().Set("Link", linkValue)
			next.ServeHTTP(w, r)
		})
	}
}

// CORS 添加跨域资源共享头部。
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Idempotency-Key, If-Match, X-Request-ID, X-Tenant-ID")
		w.Header().Set("Access-Control-Expose-Headers", "ETag, X-Trace-ID, X-Request-ID, X-Idempotent-Replayed, Deprecation, Sunset, Link")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
