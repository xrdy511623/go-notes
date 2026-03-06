package logging

import (
	"log"
	"net/http"
	"time"

	"go-notes/designpattern/dutychain/middleware"
)

// New 返回一个记录请求方法、路径和耗时的中间件。
func New() middleware.Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next(w, r)
			log.Printf("[%s] %s — %v", r.Method, r.URL.Path, time.Since(start))
		}
	}
}
