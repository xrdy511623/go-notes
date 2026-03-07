package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

/*
陷阱：测试依赖执行顺序

运行：go run .

预期行为：
  模拟三个有依赖关系的测试：
  - TestCreateOrder → 创建订单，把 ID 存到全局变量
  - TestPayOrder → 读取全局变量中的 ID，执行支付
  - TestCancelOrder → 读取全局变量中的 ID，执行取消

  当测试顺序被打乱（模拟 t.Parallel() 或随机排序），
  依赖链断裂，测试失败。

  正确做法：每个测试自己准备数据，不依赖其他测试的副作用。
*/

// Order 模拟订单
type Order struct {
	ID     int
	Status string
}

// OrderService 模拟订单服务
type OrderService struct {
	mu     sync.Mutex
	orders map[int]*Order
	nextID int
}

func NewOrderService() *OrderService {
	return &OrderService{orders: make(map[int]*Order), nextID: 1}
}

func (s *OrderService) Create(amount int) *Order {
	time.Sleep(time.Duration(rand.Intn(3)) * time.Millisecond)
	s.mu.Lock()
	defer s.mu.Unlock()
	o := &Order{ID: s.nextID, Status: "created"}
	s.orders[s.nextID] = o
	s.nextID++
	return o
}

func (s *OrderService) Pay(id int) error {
	time.Sleep(time.Duration(rand.Intn(3)) * time.Millisecond)
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[id]
	if !ok {
		return fmt.Errorf("order %d not found", id)
	}
	if o.Status != "created" {
		return fmt.Errorf("order %d status is %s, expected created", id, o.Status)
	}
	o.Status = "paid"
	return nil
}

func (s *OrderService) Cancel(id int) error {
	time.Sleep(time.Duration(rand.Intn(3)) * time.Millisecond)
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[id]
	if !ok {
		return fmt.Errorf("order %d not found", id)
	}
	o.Status = "cancelled"
	return nil
}

func main() {
	rand.New(rand.NewSource(time.Now().UnixNano()))

	fmt.Println("=== 错误做法：测试通过全局变量传递状态 ===")
	fmt.Println()

	// 模拟 3 种执行顺序
	testOrders := [][]string{
		{"TestCreate", "TestPay", "TestCancel"},
		{"TestPay", "TestCreate", "TestCancel"},
		{"TestCancel", "TestPay", "TestCreate"},
	}

	for i, order := range testOrders {
		svc := NewOrderService()
		var globalOrderID int // ❌ 全局变量

		fmt.Printf("  执行顺序 %d: %v\n", i+1, order)

		allPass := true
		for _, testName := range order {
			var result string
			switch testName {
			case "TestCreate":
				o := svc.Create(100)
				globalOrderID = o.ID
				result = fmt.Sprintf("PASS (created order %d)", o.ID)

			case "TestPay":
				if globalOrderID == 0 {
					result = "FAIL: globalOrderID=0, TestCreate hasn't run yet"
					allPass = false
				} else if err := svc.Pay(globalOrderID); err != nil {
					result = fmt.Sprintf("FAIL: %v", err)
					allPass = false
				} else {
					result = "PASS"
				}

			case "TestCancel":
				if globalOrderID == 0 {
					result = "FAIL: globalOrderID=0, TestCreate hasn't run yet"
					allPass = false
				} else if err := svc.Cancel(globalOrderID); err != nil {
					result = fmt.Sprintf("FAIL: %v", err)
					allPass = false
				} else {
					result = "PASS"
				}
			}
			fmt.Printf("    %-12s → %s\n", testName, result)
		}
		if !allPass {
			fmt.Println("    ⚠ 有测试失败！原因：依赖 TestCreate 先执行")
		}
		fmt.Println()
	}

	fmt.Println("=== 正确做法：每个测试自己准备数据 ===")
	fmt.Println()

	// 无论什么顺序都能通过
	for i, order := range testOrders {
		svc := NewOrderService()

		fmt.Printf("  执行顺序 %d: %v\n", i+1, order)

		for _, testName := range order {
			var result string
			switch testName {
			case "TestCreate":
				o := svc.Create(100)
				result = fmt.Sprintf("PASS (order %d)", o.ID)

			case "TestPay":
				// ✅ 自己创建订单，不依赖全局变量
				o := svc.Create(200)
				if err := svc.Pay(o.ID); err != nil {
					result = fmt.Sprintf("FAIL: %v", err)
				} else {
					result = fmt.Sprintf("PASS (自建 order %d 并支付)", o.ID)
				}

			case "TestCancel":
				// ✅ 自己创建订单
				o := svc.Create(300)
				if err := svc.Cancel(o.ID); err != nil {
					result = fmt.Sprintf("FAIL: %v", err)
				} else {
					result = fmt.Sprintf("PASS (自建 order %d 并取消)", o.ID)
				}
			}
			fmt.Printf("    %-12s → %s\n", testName, result)
		}
		fmt.Println("    全部通过")
		fmt.Println()
	}
}
