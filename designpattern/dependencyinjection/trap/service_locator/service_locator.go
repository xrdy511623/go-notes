// Package service_locator 演示"服务定位器"反模式。
//
// Service Locator 把所有依赖注册到一个全局注册表中，
// 消费方在运行时按名称查找依赖，而非通过构造函数显式接收。
//
// ⚠️  此代码仅作反面教材，请勿在生产环境中效仿。
//
// 问题清单：
//
//  1. 依赖隐藏在函数体内部
//     - 构造函数签名看不出需要哪些依赖
//     - 新增一个依赖不会导致编译错误，只会在运行时 panic
//
//  2. 类型安全丧失
//     - 注册表存储 any（interface{}），取出时需要类型断言
//     - 断言失败 = 运行时 panic，而非编译期报错
//
//  3. 注册顺序依赖
//     - 必须先 Register 再 Resolve，否则 panic
//     - 跨包使用时，谁先注册由调用顺序决定——脆弱且难以排查
//
//  4. 测试需要小心管理全局状态
//     - 每个测试前要 Register mock，测试后要清理
//     - 并行测试会互相干扰
//
// 对比依赖注入：
//
//	// 服务定位器——依赖隐藏
//	func NewOrderService(locator *ServiceLocator) *OrderService {
//	    repo := locator.Resolve("userRepo").(UserRepo)  // 编译器不检查
//	    ...
//	}
//
//	// 依赖注入——依赖显式
//	func NewOrderService(repo UserRepo) *OrderService {  // 编译器强制要求传入
//	    ...
//	}
package service_locator

import (
	"fmt"
	"sync"
)

// ServiceLocator 是一个全局服务注册表。
// 服务按名称注册和查找，类型信息在编译期丢失。
type ServiceLocator struct {
	mu       sync.RWMutex
	services map[string]any
}

// New 创建空的 ServiceLocator。
func New() *ServiceLocator {
	return &ServiceLocator{
		services: make(map[string]any),
	}
}

// Register 将服务注册到定位器中。
// 相同名称会被覆盖——这本身就是一个隐患。
func (sl *ServiceLocator) Register(name string, svc any) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.services[name] = svc
}

// Resolve 按名称查找服务。
// 如果服务不存在则 panic——这是服务定位器最大的问题之一：
// 编译期无法发现缺失的依赖，只有运行时才会暴露。
func (sl *ServiceLocator) Resolve(name string) any {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	svc, ok := sl.services[name]
	if !ok {
		panic(fmt.Sprintf("service_locator: service %q not registered", name))
	}
	return svc
}

// ─── 反模式示例 ──────────────────────────────────────────────

// UserRepo 模拟用户仓储接口。
type UserRepo interface {
	FindByID(id string) (string, error)
}

// OrderService 通过 ServiceLocator 获取依赖——这就是反模式。
type OrderService struct {
	locator *ServiceLocator
}

// NewOrderService 看起来只需要一个 locator，
// 但实际上它隐式依赖了 "userRepo" 和 "logger" 两个服务。
// 调用方从签名完全无法得知这些隐式依赖。
func NewOrderService(locator *ServiceLocator) *OrderService {
	return &OrderService{locator: locator}
}

// CreateOrder 演示服务定位器的典型用法。
// 类型断言失败 = 运行时 panic。
func (s *OrderService) CreateOrder(userID string) error {
	// 隐式依赖 #1：userRepo
	userRepo := s.locator.Resolve("userRepo").(UserRepo)

	name, err := userRepo.FindByID(userID)
	if err != nil {
		return fmt.Errorf("create order: %w", err)
	}

	// 隐式依赖 #2：logger
	logger := s.locator.Resolve("logger").(func(string))
	logger(fmt.Sprintf("order created for user %s", name))

	return nil
}
