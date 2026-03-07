package interfaceloss

import "net/http"

// ❌ 错误示例：装饰器导致可选接口丢失
//
// http.ResponseWriter 的底层实现通常还实现了：
//   - http.Flusher（用于 SSE / 流式响应）
//   - http.Hijacker（用于 WebSocket 升级）
//
// 如果装饰器只声明实现 http.ResponseWriter，调用方通过类型断言检查
// Flusher 时会失败，导致 SSE 等功能静默失效——不会编译报错，只在运行时暴露。
//
// 正确做法：
//   1. 实现 Unwrap() http.ResponseWriter 方法，配合 Go 1.20+ http.ResponseController
//   2. 或手动实现条件接口转发（见文件末尾注释）

type NaiveWrapper struct {
	http.ResponseWriter
	statusCode int
}

func WrapNaive(w http.ResponseWriter) *NaiveWrapper {
	return &NaiveWrapper{ResponseWriter: w, statusCode: http.StatusOK}
}

func (n *NaiveWrapper) WriteHeader(code int) {
	n.statusCode = code
	n.ResponseWriter.WriteHeader(code)
}

func (n *NaiveWrapper) StatusCode() int {
	return n.statusCode
}

// DemoInterfaceLoss 演示接口丢失问题。
// 标准库的 http.response（unexported）同时实现了 http.Flusher 和 http.Hijacker，
// 但经过 NaiveWrapper 包装后这些接口全部丢失。
func DemoInterfaceLoss(w http.ResponseWriter) {
	wrapped := WrapNaive(w)

	// 底层 w 可能是 http.Flusher（标准库的实现就是）
	if _, ok := w.(http.Flusher); ok {
		// 原始 w 支持 Flush
		_ = ok
	}

	// 但 wrapped 不是 http.Flusher——类型断言失败
	if _, ok := interface{}(wrapped).(http.Flusher); ok {
		// ← 永远不会执行，SSE 数据无法被实时推送
		_ = ok
	}
}

// ✅ 正确做法一：Unwrap（推荐，Go 1.20+）
//
//     func (n *NaiveWrapper) Unwrap() http.ResponseWriter {
//         return n.ResponseWriter
//     }
//
// http.ResponseController 会递归调用 Unwrap() 找到底层的 Flusher/Hijacker。
//
// ✅ 正确做法二：条件接口转发
//
//     type flusherWrapper struct {
//         *NaiveWrapper
//     }
//     func (f *flusherWrapper) Flush() { f.ResponseWriter.(http.Flusher).Flush() }
//
//     func WrapSmart(w http.ResponseWriter) http.ResponseWriter {
//         nw := &NaiveWrapper{ResponseWriter: w, statusCode: 200}
//         if _, ok := w.(http.Flusher); ok {
//             return &flusherWrapper{nw}
//         }
//         return nw
//     }
