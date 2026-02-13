package performance

import (
	"context"
)

var valueSink any

// 自定义 key 类型，避免跨包冲突
type benchKey int

// CreateValueChain 创建指定深度的 WithValue 链
// 模拟生产中多层 middleware 各自往 context 里塞值的场景
func CreateValueChain(depth int) (context.Context, benchKey) {
	ctx := context.Background()
	for i := range depth {
		ctx = context.WithValue(ctx, benchKey(i), i)
	}
	// 返回 context 和最底层（最早插入）的 key
	return ctx, benchKey(0)
}

// RequestMeta 将所有 request-scoped 数据打包到一个结构体中
type RequestMeta struct {
	TraceID   string
	UserID    int64
	TenantID  string
	RequestID string
	Locale    string
}

type requestMetaKey struct{}

// CreateStructValue 将所有数据打包到一个 struct 中，只做一次 WithValue
// 对比 CreateValueChain 的多次 WithValue
func CreateStructValue() context.Context {
	meta := &RequestMeta{
		TraceID:   "trace-abc-123",
		UserID:    42,
		TenantID:  "tenant-xyz",
		RequestID: "req-001",
		Locale:    "zh-CN",
	}
	return context.WithValue(context.Background(), requestMetaKey{}, meta)
}

// LookupStruct 从 context 中取出打包的结构体
func LookupStruct(ctx context.Context) *RequestMeta {
	if v, ok := ctx.Value(requestMetaKey{}).(*RequestMeta); ok {
		return v
	}
	return nil
}
