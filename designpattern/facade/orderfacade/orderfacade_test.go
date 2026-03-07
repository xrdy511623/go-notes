package orderfacade

import (
	"strings"
	"testing"

	"go-notes/designpattern/facade/inventory"
	"go-notes/designpattern/facade/notifier"
	"go-notes/designpattern/facade/payment"
)

func setup() (*Facade, *payment.Service, *inventory.Service, *notifier.Service) {
	p := payment.New()
	stock := map[string]int{"item-A": 10, "item-B": 2}
	i := inventory.New(stock)
	n := notifier.New()
	f := New(p, i, n)
	return f, p, i, n
}

func TestPlaceOrder_Success(t *testing.T) {
	f, _, _, n := setup()

	req := PlaceOrderRequest{
		OrderID:       "order-1",
		ItemID:        "item-A",
		CustomerEmail: "alice@example.com",
		Qty:           3,
		Amount:        99.99,
	}

	result, err := f.PlaceOrder(req)
	if err != nil {
		t.Fatalf("PlaceOrder unexpected error: %v", err)
	}
	if result.OrderID != "order-1" {
		t.Errorf("OrderID = %q, want %q", result.OrderID, "order-1")
	}
	if result.ChargedAmount != 99.99 {
		t.Errorf("ChargedAmount = %v, want %v", result.ChargedAmount, 99.99)
	}

	msgs := n.Sent()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(msgs))
	}
	if msgs[0].To != "alice@example.com" {
		t.Errorf("notification To = %q, want %q", msgs[0].To, "alice@example.com")
	}
	if msgs[0].Subject != "Order Confirmed" {
		t.Errorf("notification Subject = %q, want %q", msgs[0].Subject, "Order Confirmed")
	}
}

func TestPlaceOrder_InsufficientStock(t *testing.T) {
	f, _, _, n := setup()

	req := PlaceOrderRequest{
		OrderID:       "order-1",
		ItemID:        "item-B",
		CustomerEmail: "bob@example.com",
		Qty:           5,
		Amount:        50.00,
	}

	_, err := f.PlaceOrder(req)
	if err == nil {
		t.Fatal("expected error for insufficient stock")
	}
	if !strings.Contains(err.Error(), "reserve inventory") {
		t.Errorf("error should mention reserve inventory, got: %v", err)
	}

	if len(n.Sent()) != 0 {
		t.Error("no notification should be sent on inventory failure")
	}
}

func TestPlaceOrder_PaymentFails(t *testing.T) {
	f, _, i, n := setup()

	req := PlaceOrderRequest{
		OrderID:       "order-1",
		ItemID:        "item-A",
		CustomerEmail: "charlie@example.com",
		Qty:           2,
		Amount:        -10,
	}

	_, err := f.PlaceOrder(req)
	if err == nil {
		t.Fatal("expected error for invalid payment amount")
	}
	if !strings.Contains(err.Error(), "charge payment") {
		t.Errorf("error should mention charge payment, got: %v", err)
	}

	// 库存应已回滚：可以再预留相同数量
	if err := i.Reserve("order-verify", "item-A", 10); err != nil {
		t.Errorf("inventory should be fully restored after payment failure, got: %v", err)
	}

	if len(n.Sent()) != 0 {
		t.Error("no notification should be sent on payment failure")
	}
}

func TestPlaceOrder_DuplicateCharge(t *testing.T) {
	f, _, _, _ := setup()

	req := PlaceOrderRequest{
		OrderID:       "order-1",
		ItemID:        "item-A",
		CustomerEmail: "dave@example.com",
		Qty:           1,
		Amount:        10.00,
	}

	if _, err := f.PlaceOrder(req); err != nil {
		t.Fatalf("first PlaceOrder unexpected error: %v", err)
	}

	req.Qty = 1
	_, err := f.PlaceOrder(req)
	if err == nil {
		t.Fatal("expected error for duplicate order charge")
	}
}

func TestPlaceOrder_NotificationFails(t *testing.T) {
	p := payment.New()
	i := inventory.New(map[string]int{"item-A": 5})
	n := notifier.New()
	f := New(p, i, n)

	req := PlaceOrderRequest{
		OrderID:       "order-1",
		ItemID:        "item-A",
		CustomerEmail: "", // 空 email 导致通知失败
		Qty:           1,
		Amount:        20.00,
	}

	_, err := f.PlaceOrder(req)
	if err == nil {
		t.Fatal("expected error for notification failure")
	}
	if !strings.Contains(err.Error(), "send notification") {
		t.Errorf("error should mention send notification, got: %v", err)
	}

	// 支付应已退款：再次 Charge 同一 orderID 不应报重复
	if err := p.Charge("order-1", 20.00); err != nil {
		t.Errorf("payment should be refunded after notification failure, got: %v", err)
	}

	// 库存应已释放：可以再预留相同数量
	if err := i.Reserve("order-verify", "item-A", 5); err != nil {
		t.Errorf("inventory should be restored after notification failure, got: %v", err)
	}
}

func TestCancelOrder_Success(t *testing.T) {
	f, _, _, n := setup()

	req := PlaceOrderRequest{
		OrderID:       "order-1",
		ItemID:        "item-A",
		CustomerEmail: "alice@example.com",
		Qty:           2,
		Amount:        30.00,
	}

	if _, err := f.PlaceOrder(req); err != nil {
		t.Fatalf("PlaceOrder unexpected error: %v", err)
	}

	if err := f.CancelOrder("order-1"); err != nil {
		t.Fatalf("CancelOrder unexpected error: %v", err)
	}

	msgs := n.Sent()
	if len(msgs) != 2 {
		t.Fatalf("expected 2 notifications (place + cancel), got %d", len(msgs))
	}
	if msgs[1].Subject != "Order Cancelled" {
		t.Errorf("cancellation Subject = %q, want %q", msgs[1].Subject, "Order Cancelled")
	}
	if !strings.Contains(msgs[1].Body, "30.00") {
		t.Errorf("cancellation Body should contain refund amount, got: %s", msgs[1].Body)
	}
}

func TestCancelOrder_NotFound(t *testing.T) {
	f, _, _, _ := setup()

	err := f.CancelOrder("nonexistent")
	if err == nil {
		t.Fatal("expected error for cancelling unknown order")
	}
	if !strings.Contains(err.Error(), "refund payment") {
		t.Errorf("error should mention refund payment, got: %v", err)
	}
}

func TestPlaceAndCancel_StockRestored(t *testing.T) {
	p := payment.New()
	i := inventory.New(map[string]int{"item-A": 3})
	n := notifier.New()
	f := New(p, i, n)

	req := PlaceOrderRequest{
		OrderID:       "order-1",
		ItemID:        "item-A",
		CustomerEmail: "test@example.com",
		Qty:           3,
		Amount:        100.00,
	}

	if _, err := f.PlaceOrder(req); err != nil {
		t.Fatalf("PlaceOrder unexpected error: %v", err)
	}

	// 库存已全部预留，再下单应失败
	req2 := PlaceOrderRequest{
		OrderID:       "order-2",
		ItemID:        "item-A",
		CustomerEmail: "test@example.com",
		Qty:           1,
		Amount:        10.00,
	}
	if _, err := f.PlaceOrder(req2); err == nil {
		t.Fatal("expected error when stock is exhausted")
	}

	// 取消第一个订单后库存恢复
	if err := f.CancelOrder("order-1"); err != nil {
		t.Fatalf("CancelOrder unexpected error: %v", err)
	}

	if _, err := f.PlaceOrder(req2); err != nil {
		t.Fatalf("PlaceOrder after cancel should succeed, got: %v", err)
	}
}
