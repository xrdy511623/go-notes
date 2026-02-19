package main

import (
	"context"
	"fmt"
	"time"
)

/*
陷阱：在结构体中存储 context

运行：go run .

Go 官方明确建议：不要将 Context 存储在结构体中，应作为函数的第一个参数传递。

原因：
  1. context 有生命周期语义（超时、取消），存入结构体后生命周期不可控
  2. 结构体可能被长期持有，但 context 可能早已过期
  3. 同一个结构体可能在不同请求中复用，但 context 应该是 per-request 的
*/

// BadService 错误做法：将 context 存储在结构体中
type BadService struct {
	ctx    context.Context
	cancel context.CancelFunc
	name   string
}

func NewBadService(name string) *BadService {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	return &BadService{ctx: ctx, cancel: cancel, name: name}
}

func (s *BadService) Do() error {
	select {
	case <-s.ctx.Done():
		return fmt.Errorf("service %s: context expired: %w", s.name, s.ctx.Err())
	case <-time.After(50 * time.Millisecond):
		return nil
	}
}

// GoodService 正确做法：context 作为方法参数
type GoodService struct {
	name string
}

func NewGoodService(name string) *GoodService {
	return &GoodService{name: name}
}

func (s *GoodService) Do(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("service %s: context expired: %w", s.name, ctx.Err())
	case <-time.After(50 * time.Millisecond):
		return nil
	}
}

func main() {
	fmt.Println("=== 陷阱: 结构体中存储 context ===")

	// 错误用法：结构体持有的 context 可能过期
	svc := NewBadService("payment")
	fmt.Println("立即调用:", svc.Do())

	// 模拟过了一段时间后复用同一个 service 实例
	time.Sleep(150 * time.Millisecond)
	fmt.Println("150ms 后复用:", svc.Do()) // context 已过期！
	svc.cancel()

	fmt.Println("\n=== 正确做法: context 作为参数 ===")

	goodSvc := NewGoodService("payment")

	// 每次调用传入新的 context
	ctx1, cancel1 := context.WithTimeout(context.Background(), 100*time.Millisecond)
	fmt.Println("第一次调用:", goodSvc.Do(ctx1))
	cancel1()

	// 150ms 后，使用新的 context
	time.Sleep(150 * time.Millisecond)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 100*time.Millisecond)
	fmt.Println("第二次调用:", goodSvc.Do(ctx2)) // 新的 context，不受之前的影响
	cancel2()

	fmt.Println("\n总结:")
	fmt.Println("  1. Context 应该是 per-request 的，不应存入长生命周期的结构体")
	fmt.Println("  2. 正确做法: func (s *Service) Do(ctx context.Context) error")
	fmt.Println("  3. 例外: 少数场景（如 http.Request）的 context 本身就绑定到请求生命周期")
}
