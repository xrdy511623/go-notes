package auth

import (
	"net/http"

	"go-notes/designpattern/dutychain/middleware"
)

// New 返回一个验证 Authorization 头的中间件。
// 如果请求头中的 token 不匹配，返回 401 并短路责任链。
func New(token string) middleware.Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != token {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next(w, r)
		}
	}
}
