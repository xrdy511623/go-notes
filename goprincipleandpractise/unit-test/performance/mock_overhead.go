// Package mockperf 对比接口调用 vs 直接调用的性能开销
//
// 结论：接口间接调用的开销在纳秒级，对实际业务代码可忽略不计。
// 不要为了避免接口开销而牺牲可测试性。
package mockperf

// Calculator 接口
type Calculator interface {
	Add(a, b int) int
}

// RealCalculator 直接实现
type RealCalculator struct{}

func (RealCalculator) Add(a, b int) int { return a + b }

// directAdd 直接函数调用（无接口间接）
func directAdd(a, b int) int { return a + b }
