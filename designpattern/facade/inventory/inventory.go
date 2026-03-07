package inventory

import "fmt"

// Service 管理商品库存和订单预留。
// stock 记录每件商品的可用数量；reserved 按订单追踪预留明细。
type Service struct {
	stock    map[string]int
	reserved map[string]map[string]int // orderID → (itemID → qty)
}

// New 用初始库存创建库存服务。stock 会被浅拷贝以避免外部修改。
func New(stock map[string]int) *Service {
	cp := make(map[string]int, len(stock))
	for k, v := range stock {
		cp[k] = v
	}
	return &Service{
		stock:    cp,
		reserved: make(map[string]map[string]int),
	}
}

// Reserve 为订单预留指定商品。
// 扣减可用库存并记录预留明细。
func (s *Service) Reserve(orderID string, itemID string, qty int) error {
	if qty <= 0 {
		return fmt.Errorf("inventory: qty must be positive, got %d", qty)
	}
	avail, ok := s.stock[itemID]
	if !ok {
		return fmt.Errorf("inventory: item %q not found", itemID)
	}
	if avail < qty {
		return fmt.Errorf("inventory: insufficient stock for item %q (available %d, requested %d)", itemID, avail, qty)
	}

	s.stock[itemID] = avail - qty

	if s.reserved[orderID] == nil {
		s.reserved[orderID] = make(map[string]int)
	}
	s.reserved[orderID][itemID] += qty
	return nil
}

// Release 释放指定订单的所有预留库存，将库存归还。
func (s *Service) Release(orderID string) error {
	items, ok := s.reserved[orderID]
	if !ok {
		return fmt.Errorf("inventory: no reservations for order %q", orderID)
	}
	for itemID, qty := range items {
		s.stock[itemID] += qty
	}
	delete(s.reserved, orderID)
	return nil
}
