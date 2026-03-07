package proxyvsdecorator

import (
	"context"

	"go-notes/designpattern/proxy/fetcher"
)

// ❌ 容易混淆：代理和装饰器在代码结构上几乎相同
//
// 两者都是"持有接口、实现同一接口"。但意图截然不同。
//
// 判断标准只有一个：
//
//	真实主体是否总是被调用？
//	  - 总是调用 + 添加行为 → 装饰器（bufio.Reader 总是调用底层 Reader）
//	  - 可能不调用（拦截/替换）→ 代理（缓存命中时不调用、权限拒绝时不调用）
//
// 典型例子：
//
//	装饰器：CountingReader 包装 io.Reader
//	  → Read 方法中 **总是** 调用 inner.Read()，额外计数
//	  → 移除 CountingReader 后功能不受影响，只是少了计数
//
//	代理：CacheProxy 包装 Fetcher
//	  → Fetch 方法中 **可能不调用** real.Fetch()（缓存命中时跳过）
//	  → 移除 CacheProxy 后行为改变：每次都要走远程调用
//
// 正确做法：根据"是否可能拦截调用"来分类，不要被代码结构误导。

// ❌ WronglyNamedProxy 总是调用 real.Fetch 并额外计数。
// 这是装饰器行为，不应该叫 "Proxy"。
type WronglyNamedProxy struct {
	real  fetcher.Fetcher
	count int
}

func (w *WronglyNamedProxy) Fetch(ctx context.Context, key string) (string, error) {
	w.count++                     // 额外行为
	return w.real.Fetch(ctx, key) // ← 总是调用 real — 这是装饰器，不是代理
}

// ✅ CorrectProxy 可能不调用 real.Fetch（命中缓存时直接返回）。
// 这才是代理——它控制对真实主体的访问。
type CorrectProxy struct {
	real  fetcher.Fetcher
	cache map[string]string
}

func (c *CorrectProxy) Fetch(ctx context.Context, key string) (string, error) {
	if v, ok := c.cache[key]; ok {
		return v, nil // ← 不调用 real — 这是代理行为
	}
	return c.real.Fetch(ctx, key)
}
