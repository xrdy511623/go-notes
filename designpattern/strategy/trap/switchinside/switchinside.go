package switchinside

import "fmt"

// ❌ 错误示例：在方法内部用 switch 分支实现"策略选择"
//
// 这是策略模式最常见的反面。表面上代码能工作，但每新增一种支付方式，
// 都必须修改 Pay 方法——直接违反开闭原则（OCP）。
//
// 问题清单：
//  1. OCP 违反：新增支付方式 = 修改已有代码
//  2. 不可独立测试：无法单独测试某一种支付逻辑
//  3. 无法依赖注入：测试时无法 mock 特定支付方式
//  4. 臃肿膨胀：随着支付方式增多，方法体越来越长
//
// 正确做法：见 strategy/payment/ —— 每种支付方式是独立的 Strategy 实现，
// 通过 Checkout.SetStrategy() 在运行时注入，新增支付方式不改任何已有代码。

type Checkout struct {
	Method string
}

func (c *Checkout) Pay(amount float64) (string, error) {
	switch c.Method {
	case "alipay":
		return fmt.Sprintf("ALIPAY-%.2f", amount), nil
	case "wechat":
		return fmt.Sprintf("WECHAT-%.2f", amount), nil
	case "creditcard":
		return fmt.Sprintf("CC-%.2f", amount), nil
	// ❌ 每新增一种支付方式，都要在这里加一个 case
	// case "paypal": ...
	// case "crypto": ...
	default:
		return "", fmt.Errorf("unknown payment method: %s", c.Method)
	}
}
