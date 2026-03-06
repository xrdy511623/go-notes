package recovery

import (
	"log"
	"net/http"

	"go-notes/designpattern/dutychain/middleware"
)

// New 返回一个捕获 panic 的中间件。
// 将 panic 转换为 500 响应，防止单个请求的异常导致整个服务崩溃。
func New() middleware.Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Printf("panic recovered: %v", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next(w, r)
		}
	}
}
