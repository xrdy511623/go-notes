package nofallback

import "errors"

// ❌ 反模式一：所有错误都触发熔断
//
// 熔断器应该只对"基础设施错误"计数（超时、连接拒绝、5xx）。
// 业务错误（参数校验失败 400、资源不存在 404）不应该触发熔断。
//
// 错误示例：
//
//	breaker.Do(func() error {
//	    resp, err := http.Get(url)
//	    if err != nil {
//	        return err  // ✅ 网络错误 — 应该触发熔断
//	    }
//	    if resp.StatusCode == 404 {
//	        return fmt.Errorf("not found")  // ❌ 业务错误 — 不应触发熔断
//	    }
//	    return nil
//	})
//
// 正确做法：区分可熔断错误和业务错误。
// sony/gobreaker 通过 IsSuccessful func(err error) bool 回调实现。
// 本示例通过 BreakableError 标记实现。

// BreakableError 标记一个错误应该触发熔断计数。
// 只有被此类型包装的错误才计入连续失败统计。
type BreakableError struct {
	Err error
}

func (e *BreakableError) Error() string { return e.Err.Error() }
func (e *BreakableError) Unwrap() error { return e.Err }

// IsBreakable 判断 err 是否应触发熔断。
func IsBreakable(err error) bool {
	var be *BreakableError
	return errors.As(err, &be)
}

// ❌ 反模式二：熔断打开后只返回错误，没有降级
//
//	value, err := fetchFromDB(key)   // 被 breaker 保护
//	if err != nil {
//	    return "", err  // ❌ 直接返回 500，用户看到白屏
//	}
//
// ✅ 正确做法：提供降级逻辑（返回缓存/默认值/静态页面）
//
//	value, err := fetchFromDB(key)
//	if err != nil {
//	    if cached, ok := localCache.Get(key); ok {
//	        return cached, nil   // ← 返回过期但可用的缓存
//	    }
//	    return defaultValue, nil // ← 返回默认值，保证页面可展示
//	}
//
// 降级策略的选择：
//   - 缓存降级：返回上一次成功的结果（适合读场景）
//   - 默认值降级：返回空列表/默认配置（适合配置中心）
//   - 静态降级：返回预置的静态页面/提示（适合前端页面）
//   - 排队降级：将请求放入队列稍后处理（适合写场景）
