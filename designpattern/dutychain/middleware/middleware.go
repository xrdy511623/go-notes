package middleware

import "net/http"

// Middleware 将一个 http.HandlerFunc 转换为另一个。
// 每个 Middleware 可以在调用 next 之前/之后执行逻辑，
// 也可以通过不调用 next 来短路（中断）责任链。
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Chain 将多个 Middleware 组合成单个 Middleware。
// 执行顺序与传入顺序一致：Chain(A, B, C)(handler) 的请求路径为 A → B → C → handler。
// Chain 本身也是 Middleware，因此链可以嵌套组合。
func Chain(middlewares ...Middleware) Middleware {
	return func(final http.HandlerFunc) http.HandlerFunc {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}

// Apply 是 Chain 的便捷方法：将多个 Middleware 依次应用到最终处理函数。
func Apply(handler http.HandlerFunc, middlewares ...Middleware) http.HandlerFunc {
	return Chain(middlewares...)(handler)
}
