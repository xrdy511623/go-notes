package interfacevsgenerics

/*
陷阱：混淆接口和泛型的使用场景

问题说明：
  接口和泛型都能实现"一份代码处理多种类型"，但机制不同：

  - 接口：运行时多态，不同类型有不同行为
  - 泛型：编译时多态，不同类型共享相同逻辑

  常见错误：
  1. 不同类型逻辑不同时用泛型 → 被迫在泛型函数内做类型判断
  2. 纯数据处理时用 interface{} → 丢失类型安全，到处类型断言
  3. 需要依赖注入时用泛型 → 无法运行时替换实现

正确做法：
  接口抽象行为，泛型抽象类型。
  - 不同类型、不同逻辑 → 接口
  - 不同类型、相同逻辑 → 泛型
*/

import "fmt"

// ======= 错误示例 1：该用接口的地方用了泛型 =======

// ❌ 不同存储的逻辑完全不同，泛型无法表达这种差异
// 被迫在泛型函数内做类型判断，完全失去了泛型的意义
type Storage interface {
	Save(key string, data []byte) error
}

type MemoryStorage struct {
	data map[string][]byte
}

func (m *MemoryStorage) Save(key string, data []byte) error {
	if m.data == nil {
		m.data = make(map[string][]byte)
	}
	m.data[key] = data
	return nil
}

type FileStorage struct {
	basePath string
}

func (f *FileStorage) Save(key string, data []byte) error {
	// 实际会写文件，这里省略
	_ = f.basePath
	return nil
}

// ✅ 正确：用接口，不同实现各自封装逻辑
func SaveData(s Storage, key string, data []byte) error {
	return s.Save(key, data)
}

// ======= 错误示例 2：该用泛型的地方用了 interface{} =======

// ❌ 用 interface{} 实现通用容器：类型不安全
type BadStack struct {
	items []interface{}
}

func (s *BadStack) Push(item interface{}) {
	s.items = append(s.items, item)
}

func (s *BadStack) Pop() interface{} {
	if len(s.items) == 0 {
		return nil
	}
	item := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return item
}

// ✅ 正确：用泛型实现通用容器：类型安全
type GoodStack[T any] struct {
	items []T
}

func (s *GoodStack[T]) Push(item T) {
	s.items = append(s.items, item)
}

func (s *GoodStack[T]) Pop() (T, bool) {
	var zero T
	if len(s.items) == 0 {
		return zero, false
	}
	item := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return item, true
}

// DemonstrateProblem 演示错误选择的后果
func DemonstrateProblem() {
	fmt.Println("=== 混淆接口和泛型 ===")
	fmt.Println()

	// interface{} 容器的问题：可以放入任意类型，取出时需要断言
	bad := &BadStack{}
	bad.Push(1)
	bad.Push("hello") // 编译不报错！类型混乱
	val := bad.Pop()
	// 需要类型断言，如果断言错误就 panic
	fmt.Printf("interface{} 栈弹出: %v (需要类型断言才能使用)\n", val)

	// 泛型容器：编译时类型安全
	good := &GoodStack[int]{}
	good.Push(1)
	// good.Push("hello") // 编译错误：cannot use "hello" as int
	v, ok := good.Pop()
	fmt.Printf("泛型栈弹出: %d, ok=%v (类型安全，无需断言)\n", v, ok)

	fmt.Println()
	fmt.Println("决策指南：")
	fmt.Println("  不同类型、不同逻辑 → 接口（如存储抽象）")
	fmt.Println("  不同类型、相同逻辑 → 泛型（如容器、算法）")
	fmt.Println("  需要依赖注入/mock → 接口")
	fmt.Println("  需要运算符操作     → 泛型")
}
