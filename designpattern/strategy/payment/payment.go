package payment

import (
	"errors"
	"fmt"
)

// Strategy 定义支付算法的统一接口。
// 不同支付方式实现该接口，Checkout 在运行时选择使用哪一种。
//
// 这是接口策略的经典形式，与 http.RoundTripper 思路一致：
// http.Transport 是默认的 RoundTripper 实现，测试时可以替换为 mock。
type Strategy interface {
	Pay(amount float64) (transactionID string, err error)
}

// StrategyFunc 是 Strategy 的函数类型适配器。
// 与 http.HandlerFunc → http.Handler 完全同一模式：
// 任何签名匹配的函数都可以直接当作 Strategy 使用。
//
// 典型场景：测试时传入匿名函数做 mock，无需定义额外的 struct。
type StrategyFunc func(amount float64) (string, error)

func (f StrategyFunc) Pay(amount float64) (string, error) {
	return f(amount)
}

// --- 具体策略实现 ---

// Alipay 支付宝支付策略。
type Alipay struct {
	MerchantID string
}

func (a *Alipay) Pay(amount float64) (string, error) {
	if amount <= 0 {
		return "", errors.New("alipay: amount must be positive")
	}
	return fmt.Sprintf("ALIPAY-%s-%.2f", a.MerchantID, amount), nil
}

// WechatPay 微信支付策略。
type WechatPay struct {
	AppID string
}

func (w *WechatPay) Pay(amount float64) (string, error) {
	if amount <= 0 {
		return "", errors.New("wechat: amount must be positive")
	}
	return fmt.Sprintf("WECHAT-%s-%.2f", w.AppID, amount), nil
}

// CreditCard 信用卡支付策略。
type CreditCard struct {
	Gateway string
}

func (c *CreditCard) Pay(amount float64) (string, error) {
	if amount <= 0 {
		return "", errors.New("creditcard: amount must be positive")
	}
	return fmt.Sprintf("CC-%s-%.2f", c.Gateway, amount), nil
}

// Checkout 持有当前的支付策略，将实际的支付行为委托给它。
// 调用方可以在运行时通过 SetStrategy 切换策略，而无需修改 Checkout 的任何代码。
type Checkout struct {
	strategy Strategy
}

// NewCheckout 创建一个使用指定支付策略的 Checkout。
func NewCheckout(s Strategy) *Checkout {
	return &Checkout{strategy: s}
}

// SetStrategy 在运行时替换支付策略。
func (c *Checkout) SetStrategy(s Strategy) {
	c.strategy = s
}

// Pay 委托给当前策略执行支付。
func (c *Checkout) Pay(amount float64) (string, error) {
	if c.strategy == nil {
		return "", errors.New("checkout: no payment strategy set")
	}
	return c.strategy.Pay(amount)
}
