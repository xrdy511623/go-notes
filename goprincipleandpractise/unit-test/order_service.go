package unittest

import (
	"errors"
	"fmt"
	"time"
)

// ---------- 领域模型 ----------

// Product 表示商品
type Product struct {
	ID    string
	Name  string
	Price float64
	Stock int
}

// Coupon 表示优惠券
type Coupon struct {
	Code      string
	Discount  float64   // 折扣金额
	MinAmount float64   // 最低消费金额
	ExpiresAt time.Time // 过期时间
	Used      bool      // 是否已使用
}

// Order 表示订单
type Order struct {
	ID         string
	Items      []OrderItem
	CouponCode string
	Total      float64
}

// OrderItem 表示订单中的一个商品行
type OrderItem struct {
	ProductID string
	Quantity  int
	Price     float64
}

// ---------- 接口定义 ----------

// ProductRepo 商品仓库
type ProductRepo interface {
	GetByID(id string) (*Product, error)
}

// CouponRepo 优惠券仓库
type CouponRepo interface {
	GetByCode(code string) (*Coupon, error)
}

// OrderRepo 订单仓库
type OrderRepo interface {
	Save(order *Order) error
}

// Notifier 通知服务
type Notifier interface {
	SendOrderConfirmation(orderID string) error
}

// ---------- 业务层 ----------

// OrderService 订单服务，通过构造函数注入所有依赖。
// 方法拆分为 ValidateStock、ApplyCoupon、CalculateTotal、PlaceOrder，
// 每个方法职责单一，便于独立测试。
type OrderService struct {
	products ProductRepo
	coupons  CouponRepo
	orders   OrderRepo
	notifier Notifier
}

// NewOrderService 创建订单服务实例
func NewOrderService(p ProductRepo, c CouponRepo, o OrderRepo, n Notifier) *OrderService {
	return &OrderService{products: p, coupons: c, orders: o, notifier: n}
}

// 常见订单错误
var (
	ErrEmptyItems        = errors.New("order must have at least one item")
	ErrInvalidQuantity   = errors.New("quantity must be positive")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrCouponExpired     = errors.New("coupon has expired")
	ErrCouponUsed        = errors.New("coupon already used")
	ErrBelowMinAmount    = errors.New("order total below coupon minimum amount")
)

// ValidateStock 检查所有商品库存是否充足
func (s *OrderService) ValidateStock(items []OrderItem) error {
	if len(items) == 0 {
		return ErrEmptyItems
	}
	for _, item := range items {
		if item.Quantity <= 0 {
			return ErrInvalidQuantity
		}
		product, err := s.products.GetByID(item.ProductID)
		if err != nil {
			return fmt.Errorf("check stock for %s: %w", item.ProductID, err)
		}
		if product.Stock < item.Quantity {
			return fmt.Errorf("product %s: %w", item.ProductID, ErrInsufficientStock)
		}
	}
	return nil
}

// ApplyCoupon 验证并返回优惠券折扣金额，couponCode 为空则返回 0。
func (s *OrderService) ApplyCoupon(couponCode string, subtotal float64) (float64, error) {
	if couponCode == "" {
		return 0, nil
	}
	coupon, err := s.coupons.GetByCode(couponCode)
	if err != nil {
		return 0, fmt.Errorf("get coupon %s: %w", couponCode, err)
	}
	if coupon.Used {
		return 0, ErrCouponUsed
	}
	if time.Now().After(coupon.ExpiresAt) {
		return 0, ErrCouponExpired
	}
	if subtotal < coupon.MinAmount {
		return 0, ErrBelowMinAmount
	}
	return coupon.Discount, nil
}

// CalculateTotal 根据商品和折扣计算最终金额
func (s *OrderService) CalculateTotal(items []OrderItem, discount float64) float64 {
	var subtotal float64
	for _, item := range items {
		subtotal += item.Price * float64(item.Quantity)
	}
	total := subtotal - discount
	if total < 0 {
		total = 0
	}
	return total
}

// PlaceOrder 执行完整下单流程：校验库存 → 计算金额 → 应用优惠券 → 保存 → 通知
func (s *OrderService) PlaceOrder(items []OrderItem, couponCode string) (*Order, error) {
	if err := s.ValidateStock(items); err != nil {
		return nil, err
	}

	var subtotal float64
	for _, item := range items {
		subtotal += item.Price * float64(item.Quantity)
	}

	discount, err := s.ApplyCoupon(couponCode, subtotal)
	if err != nil {
		return nil, err
	}

	total := s.CalculateTotal(items, discount)

	order := &Order{
		ID:         fmt.Sprintf("ord_%d", time.Now().UnixNano()),
		Items:      items,
		CouponCode: couponCode,
		Total:      total,
	}

	if err := s.orders.Save(order); err != nil {
		return nil, fmt.Errorf("save order: %w", err)
	}

	// 通知失败不影响下单成功（最终一致性）
	_ = s.notifier.SendOrderConfirmation(order.ID)

	return order, nil
}
