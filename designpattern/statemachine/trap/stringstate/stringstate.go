package stringstate

// ❌ 反模式：用裸字符串表示状态和事件
//
// 裸字符串没有编译期检查，拼写错误只能在运行时发现：
//
//	current := "idle"
//	if current == "idel" {   // ← 拼写错误，永远为 false
//	    // 永远不会执行
//	}
//
//	fsm.Fire("colse")  // ← 拼写错误，运行时返回 ErrUnknownEvent
//
// ✅ 正确做法：使用类型化常量
//
//	const Idle fsm.State = "idle"
//	const Close fsm.Event = "close"
//
//	if m.Current() == Idle { ... }  // ← 编译器保证 Idle 存在
//	m.Fire(Close)                   // ← 编译器保证 Close 存在
//
// 更进一步：用 iota + Stringer 生成状态名称：
//
//	type OrderState int
//	const (
//	    Created OrderState = iota
//	    Paid
//	    Shipped
//	)
//
// 但 looplab/fsm 和本项目选择 string 类型是因为：
//   - 字符串可读性更好（日志、调试直接输出）
//   - 类型别名（type State string）提供了部分类型安全
//   - Go 没有 enum，iota 方案需要额外的 Stringer 代码生成

// ❌ BadFire 展示裸字符串的危险：拼写错误在编译期不可见。
func BadFire(state string) string {
	switch state {
	case "idle":
		return "actve" // ← 拼写错误：少了 i
	case "active":
		return "closed"
	default:
		return state
	}
}
