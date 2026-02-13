package performance

import (
	"context"
	"time"
)

var cancelSink context.Context

// CreateCancelTree 创建一个有 N 个子节点的取消树
// 返回根 context 和 cancel 函数
func CreateCancelTree(children int) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	for range children {
		child, childCancel := context.WithCancel(ctx)
		_ = child
		_ = childCancel
	}
	return ctx, cancel
}

// CreateDeepCancelChain 创建一条深度为 N 的取消链（非树，线性）
// parent → child1 → child2 → ... → childN
func CreateDeepCancelChain(depth int) (context.Context, context.CancelFunc) {
	root, rootCancel := context.WithCancel(context.Background())
	ctx := root
	for range depth {
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		_ = cancel
	}
	return root, rootCancel
}

// ContextCreationCancel 创建 WithCancel 的开销
func ContextCreationCancel() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}

// ContextCreationTimeout 创建 WithTimeout 的开销
func ContextCreationTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Hour)
}

// ContextCreationValue 创建 WithValue 的开销
func ContextCreationValue() context.Context {
	return context.WithValue(context.Background(), benchKey(0), "value")
}

// ContextCreationCancelCause 创建 WithCancelCause 的开销（Go 1.20+）
func ContextCreationCancelCause() (context.Context, context.CancelCauseFunc) {
	return context.WithCancelCause(context.Background())
}
