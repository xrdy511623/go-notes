// Package godfacade 演示门面模式的典型反例 —— "上帝门面"。
//
// 反模式特征：
//   - 违反单一职责（SRP）：门面不再只做协调，而是自己实现了支付、库存、通知的所有逻辑。
//   - 无法独立测试子系统：支付逻辑、库存逻辑、通知逻辑全部耦合在一个 struct 里，
//     无法对它们分别写单元测试。
//   - 任何子系统的变化都要修改门面：比如"支付增加币种"需要改 GodFacade，
//     "库存增加仓库维度"也需要改 GodFacade，变更影响面不可控。
//   - 所有本应属于子系统的字段都堆在同一个 struct 里，随着业务增长会迅速膨胀。
//
// 正确做法：门面只持有子系统的引用并做编排，具体业务逻辑由各子系统自行实现。
// 参见 orderfacade/ 中的正确实现。
package godfacade

import "fmt"

// GodFacade 把所有子系统的状态和逻辑都内联到自身。
// 这是一个反面教材，不要在生产代码中使用。
type GodFacade struct {
	charged  map[string]float64
	stock    map[string]int
	reserved map[string]map[string]int
	emails   []string
}

// New 创建一个 GodFacade。
func New(stock map[string]int) *GodFacade {
	cp := make(map[string]int, len(stock))
	for k, v := range stock {
		cp[k] = v
	}
	return &GodFacade{
		charged:  make(map[string]float64),
		stock:    cp,
		reserved: make(map[string]map[string]int),
	}
}

// PlaceOrder 内联了支付扣款、库存预留、通知发送的所有逻辑。
// 一旦任何子系统需要调整（如支付增加幂等校验、库存增加多仓支持），
// 都必须直接修改这个方法，违反开闭原则。
func (g *GodFacade) PlaceOrder(orderID, itemID, email string, qty int, amount float64) error {
	// --- 内联库存逻辑（本应在 inventory.Service 中） ---
	avail, ok := g.stock[itemID]
	if !ok {
		return fmt.Errorf("godfacade: item %q not found", itemID)
	}
	if avail < qty {
		return fmt.Errorf("godfacade: insufficient stock for %q", itemID)
	}
	g.stock[itemID] -= qty
	if g.reserved[orderID] == nil {
		g.reserved[orderID] = make(map[string]int)
	}
	g.reserved[orderID][itemID] += qty

	// --- 内联支付逻辑（本应在 payment.Service 中） ---
	if amount <= 0 {
		g.stock[itemID] += qty
		delete(g.reserved, orderID)
		return fmt.Errorf("godfacade: invalid amount %v", amount)
	}
	if _, dup := g.charged[orderID]; dup {
		g.stock[itemID] += qty
		delete(g.reserved, orderID)
		return fmt.Errorf("godfacade: duplicate charge for %q", orderID)
	}
	g.charged[orderID] = amount

	// --- 内联通知逻辑（本应在 notifier.Service 中） ---
	if email == "" {
		delete(g.charged, orderID)
		g.stock[itemID] += qty
		delete(g.reserved, orderID)
		return fmt.Errorf("godfacade: empty email")
	}
	g.emails = append(g.emails, fmt.Sprintf("to=%s order=%s", email, orderID))

	return nil
}
