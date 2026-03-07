package payment

import "fmt"

// Service 处理订单支付和退款。
// charged 记录每笔订单已扣款的金额，防止重复扣款。
type Service struct {
	charged map[string]float64
}

// New 创建一个新的支付服务实例。
func New() *Service {
	return &Service{
		charged: make(map[string]float64),
	}
}

// Charge 对指定订单扣款。
// amount 必须大于 0，且同一订单不允许重复扣款。
func (s *Service) Charge(orderID string, amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("payment: amount must be positive, got %v", amount)
	}
	if _, ok := s.charged[orderID]; ok {
		return fmt.Errorf("payment: order %q already charged", orderID)
	}
	s.charged[orderID] = amount
	return nil
}

// Refund 退还指定订单的已扣款金额。
// 如果订单未被扣款，返回错误。
func (s *Service) Refund(orderID string) (float64, error) {
	amount, ok := s.charged[orderID]
	if !ok {
		return 0, fmt.Errorf("payment: order %q not found", orderID)
	}
	delete(s.charged, orderID)
	return amount, nil
}
