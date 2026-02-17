package unittest

import (
	"errors"
	"testing"
	"time"
)

// ---------- Stub 实现 ----------

type stubProductRepo struct {
	products map[string]*Product
	err      error
}

func (s *stubProductRepo) GetByID(id string) (*Product, error) {
	if s.err != nil {
		return nil, s.err
	}
	p, ok := s.products[id]
	if !ok {
		return nil, errors.New("product not found")
	}
	return p, nil
}

type stubCouponRepo struct {
	coupons map[string]*Coupon
	err     error
}

func (s *stubCouponRepo) GetByCode(code string) (*Coupon, error) {
	if s.err != nil {
		return nil, s.err
	}
	c, ok := s.coupons[code]
	if !ok {
		return nil, errors.New("coupon not found")
	}
	return c, nil
}

type stubOrderRepo struct {
	saved []*Order
	err   error
}

func (s *stubOrderRepo) Save(order *Order) error {
	if s.err != nil {
		return s.err
	}
	s.saved = append(s.saved, order)
	return nil
}

type stubNotifier struct {
	sent []string
	err  error
}

func (s *stubNotifier) SendOrderConfirmation(orderID string) error {
	if s.err != nil {
		return s.err
	}
	s.sent = append(s.sent, orderID)
	return nil
}

// ---------- ValidateStock 测试 ----------

func TestOrderService_ValidateStock(t *testing.T) {
	tests := []struct {
		name     string
		products map[string]*Product
		items    []OrderItem
		wantErr  error
	}{
		{
			name: "库存充足",
			products: map[string]*Product{
				"p1": {ID: "p1", Stock: 10},
				"p2": {ID: "p2", Stock: 5},
			},
			items: []OrderItem{
				{ProductID: "p1", Quantity: 3, Price: 10},
				{ProductID: "p2", Quantity: 2, Price: 20},
			},
		},
		{
			name:     "空商品列表",
			products: map[string]*Product{},
			items:    []OrderItem{},
			wantErr:  ErrEmptyItems,
		},
		{
			name: "库存不足",
			products: map[string]*Product{
				"p1": {ID: "p1", Stock: 2},
			},
			items: []OrderItem{
				{ProductID: "p1", Quantity: 5, Price: 10},
			},
			wantErr: ErrInsufficientStock,
		},
		{
			name: "数量为零",
			products: map[string]*Product{
				"p1": {ID: "p1", Stock: 10},
			},
			items: []OrderItem{
				{ProductID: "p1", Quantity: 0, Price: 10},
			},
			wantErr: ErrInvalidQuantity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewOrderService(
				&stubProductRepo{products: tt.products},
				&stubCouponRepo{},
				&stubOrderRepo{},
				&stubNotifier{},
			)
			err := svc.ValidateStock(tt.items)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("got error %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// ---------- ApplyCoupon 测试 ----------

func TestOrderService_ApplyCoupon(t *testing.T) {
	future := time.Now().Add(24 * time.Hour)
	past := time.Now().Add(-24 * time.Hour)

	tests := []struct {
		name     string
		coupons  map[string]*Coupon
		code     string
		subtotal float64
		want     float64
		wantErr  error
	}{
		{
			name:     "无优惠券",
			coupons:  map[string]*Coupon{},
			code:     "",
			subtotal: 100,
			want:     0,
		},
		{
			name: "有效优惠券",
			coupons: map[string]*Coupon{
				"SAVE10": {Code: "SAVE10", Discount: 10, MinAmount: 50, ExpiresAt: future},
			},
			code:     "SAVE10",
			subtotal: 100,
			want:     10,
		},
		{
			name: "过期优惠券",
			coupons: map[string]*Coupon{
				"OLD": {Code: "OLD", Discount: 10, MinAmount: 0, ExpiresAt: past},
			},
			code:     "OLD",
			subtotal: 100,
			wantErr:  ErrCouponExpired,
		},
		{
			name: "已使用优惠券",
			coupons: map[string]*Coupon{
				"USED": {Code: "USED", Discount: 10, MinAmount: 0, ExpiresAt: future, Used: true},
			},
			code:     "USED",
			subtotal: 100,
			wantErr:  ErrCouponUsed,
		},
		{
			name: "未达到最低金额",
			coupons: map[string]*Coupon{
				"MIN100": {Code: "MIN100", Discount: 10, MinAmount: 100, ExpiresAt: future},
			},
			code:     "MIN100",
			subtotal: 50,
			wantErr:  ErrBelowMinAmount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewOrderService(
				&stubProductRepo{},
				&stubCouponRepo{coupons: tt.coupons},
				&stubOrderRepo{},
				&stubNotifier{},
			)
			got, err := svc.ApplyCoupon(tt.code, tt.subtotal)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("got error %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("discount = %v, want %v", got, tt.want)
			}
		})
	}
}

// ---------- CalculateTotal 测试 ----------

func TestOrderService_CalculateTotal(t *testing.T) {
	svc := NewOrderService(nil, nil, nil, nil)

	tests := []struct {
		name     string
		items    []OrderItem
		discount float64
		want     float64
	}{
		{
			name: "正常计算",
			items: []OrderItem{
				{Price: 10, Quantity: 2},
				{Price: 20, Quantity: 1},
			},
			discount: 5,
			want:     35,
		},
		{
			name:     "空列表",
			items:    []OrderItem{},
			discount: 0,
			want:     0,
		},
		{
			name: "折扣大于总额",
			items: []OrderItem{
				{Price: 10, Quantity: 1},
			},
			discount: 20,
			want:     0, // 不会变成负数
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.CalculateTotal(tt.items, tt.discount)
			if got != tt.want {
				t.Errorf("total = %v, want %v", got, tt.want)
			}
		})
	}
}

// ---------- PlaceOrder 集成测试 ----------

func TestOrderService_PlaceOrder(t *testing.T) {
	future := time.Now().Add(24 * time.Hour)

	orderRepo := &stubOrderRepo{}
	notifier := &stubNotifier{}
	svc := NewOrderService(
		&stubProductRepo{products: map[string]*Product{
			"p1": {ID: "p1", Stock: 10, Price: 100},
		}},
		&stubCouponRepo{coupons: map[string]*Coupon{
			"SAVE10": {Code: "SAVE10", Discount: 10, MinAmount: 50, ExpiresAt: future},
		}},
		orderRepo,
		notifier,
	)

	items := []OrderItem{{ProductID: "p1", Quantity: 2, Price: 100}}
	order, err := svc.PlaceOrder(items, "SAVE10")
	if err != nil {
		t.Fatalf("PlaceOrder failed: %v", err)
	}

	if order.Total != 190 { // 2*100 - 10
		t.Errorf("order.Total = %v, want 190", order.Total)
	}
	if len(orderRepo.saved) != 1 {
		t.Errorf("orders saved = %d, want 1", len(orderRepo.saved))
	}
	if len(notifier.sent) != 1 {
		t.Errorf("notifications sent = %d, want 1", len(notifier.sent))
	}
}
