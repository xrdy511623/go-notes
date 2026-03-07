package responsewriter

import "net/http"

// ResponseRecorder 包装 http.ResponseWriter，捕获状态码和响应体大小。
// 这是 Go Web 开发中最常见的装饰器应用：
//   - 日志中间件需要记录响应状态码
//   - 监控中间件需要统计响应体大小
//   - 标准库的 http.ResponseWriter 不提供读取这些信息的方法
//
// 标准库 httptest.ResponseRecorder 也是同一思路，但它把响应体缓存到内存中，
// 不适合生产环境。本实现是零缓冲的——数据直接写入底层 ResponseWriter。
type ResponseRecorder struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
	wroteHeader  bool
}

// NewResponseRecorder 返回一个包装了 w 的 ResponseRecorder。
// 如果 WriteHeader 未被显式调用，默认状态码为 200。
func NewResponseRecorder(w http.ResponseWriter) *ResponseRecorder {
	return &ResponseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (r *ResponseRecorder) WriteHeader(code int) {
	if r.wroteHeader {
		return
	}
	r.statusCode = code
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(code)
}

func (r *ResponseRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytesWritten += int64(n)
	return n, err
}

// StatusCode 返回响应状态码。
func (r *ResponseRecorder) StatusCode() int {
	return r.statusCode
}

// BytesWritten 返回已写入的响应体字节数。
func (r *ResponseRecorder) BytesWritten() int64 {
	return r.bytesWritten
}

// Unwrap 返回底层的 http.ResponseWriter。
// Go 1.20+ 的 http.ResponseController 通过 Unwrap 递归发现底层 Writer 的可选接口
// （如 http.Flusher、http.Hijacker），避免装饰器导致的接口丢失问题。
func (r *ResponseRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}
