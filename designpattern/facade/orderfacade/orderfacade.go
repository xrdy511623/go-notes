package orderfacade

import (
	"fmt"

	"go-notes/designpattern/facade/inventory"
	"go-notes/designpattern/facade/notifier"
	"go-notes/designpattern/facade/payment"
)

// PlaceOrderRequest 包含下单所需的全部信息。
type PlaceOrderRequest struct {
	OrderID       string
	ItemID        string
	CustomerEmail string
	Qty           int
	Amount        float64
}

// PlaceOrderResult 包含下单成功后的摘要信息。
type PlaceOrderResult struct {
	OrderID       string
	ChargedAmount float64
}

// Facade 是订单流程的门面，协调库存、支付、通知三个子系统。
// 调用方只需与 Facade 交互，无需了解各子系统的调用顺序和回滚逻辑。
type Facade struct {
	payment   *payment.Service
	inventory *inventory.Service
	notifier  *notifier.Service
}

// New 创建一个 Facade 实例。三个子系统通过构造函数注入，便于测试。
func New(p *payment.Service, i *inventory.Service, n *notifier.Service) *Facade {
	return &Facade{
		payment:   p,
		inventory: i,
		notifier:  n,
	}
}

// PlaceOrder 执行完整的下单流程：预留库存 → 扣款 → 发送通知。
// 任何步骤失败都会回滚已完成的操作，保证不会出现"扣了款却没锁库存"的不一致状态。
func (f *Facade) PlaceOrder(req PlaceOrderRequest) (PlaceOrderResult, error) {
	// 1. 预留库存
	if err := f.inventory.Reserve(req.OrderID, req.ItemID, req.Qty); err != nil {
		return PlaceOrderResult{}, fmt.Errorf("orderfacade: reserve inventory: %w", err)
	}

	// 2. 扣款（失败则释放库存）
	if err := f.payment.Charge(req.OrderID, req.Amount); err != nil {
		f.inventory.Release(req.OrderID)
		return PlaceOrderResult{}, fmt.Errorf("orderfacade: charge payment: %w", err)
	}

	// 3. 发送通知（失败则退款并释放库存）
	err := f.notifier.Send(notifier.Message{
		To:      req.CustomerEmail,
		Subject: "Order Confirmed",
		Body:    fmt.Sprintf("Your order %s has been placed successfully.", req.OrderID),
	})
	if err != nil {
		f.payment.Refund(req.OrderID)
		f.inventory.Release(req.OrderID)
		return PlaceOrderResult{}, fmt.Errorf("orderfacade: send notification: %w", err)
	}

	return PlaceOrderResult{
		OrderID:       req.OrderID,
		ChargedAmount: req.Amount,
	}, nil
}

// CancelOrder 执行取消订单流程：退款 → 释放库存 → 发送取消通知。
func (f *Facade) CancelOrder(orderID string) error {
	amount, err := f.payment.Refund(orderID)
	if err != nil {
		return fmt.Errorf("orderfacade: refund payment: %w", err)
	}

	if err := f.inventory.Release(orderID); err != nil {
		return fmt.Errorf("orderfacade: release inventory: %w", err)
	}

	err = f.notifier.Send(notifier.Message{
		To:      "customer",
		Subject: "Order Cancelled",
		Body:    fmt.Sprintf("Your order %s has been cancelled. Refund: %.2f", orderID, amount),
	})
	if err != nil {
		return fmt.Errorf("orderfacade: send cancellation notification: %w", err)
	}

	return nil
}
