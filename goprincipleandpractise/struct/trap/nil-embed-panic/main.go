package main

import "fmt"

/*
陷阱：嵌入指针类型为 nil 时，调用方法导致 panic

运行：go run .

预期行为：
  嵌入一个指针类型（如 *Inner）时，如果该指针为 nil，
  通过外层 struct 调用被提升的方法会触发 nil pointer dereference panic。
  这是因为方法提升后，调用时仍需要解引用指针来获取 receiver。

  与嵌入值类型不同：值类型嵌入总是有零值，不会 panic。
*/

type Inner struct {
	Value int
}

func (i *Inner) GetValue() int {
	return i.Value
}

func (i *Inner) String() string {
	return fmt.Sprintf("Inner{Value: %d}", i.Value)
}

// Outer 嵌入 *Inner（指针类型）
type Outer struct {
	*Inner // 注意是指针嵌入
	Name   string
}

// SafeOuter 嵌入 Inner（值类型）
type SafeOuter struct {
	Inner // 值类型嵌入，零值安全
	Name  string
}

func main() {
	fmt.Println("=== 演示1: 值类型嵌入 — 零值安全 ===")
	safe := SafeOuter{Name: "safe"}
	fmt.Printf("  safe.GetValue() = %d（零值，不会 panic）\n", safe.GetValue())
	fmt.Printf("  safe.String() = %s\n", safe.String())

	fmt.Println("\n=== 演示2: 指针类型嵌入 — 正确初始化 ===")
	good := Outer{Inner: &Inner{Value: 42}, Name: "good"}
	fmt.Printf("  good.GetValue() = %d\n", good.GetValue())

	fmt.Println("\n=== 演示3: 指针类型嵌入 — nil 时 panic ===")
	bad := Outer{Name: "bad"} // Inner 为 nil
	fmt.Printf("  bad.Inner == nil: %v\n", bad.Inner == nil)

	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("  panic recovered: %v\n", r)
			}
		}()
		_ = bad.GetValue() // panic: nil pointer dereference
	}()

	fmt.Println("\n总结:")
	fmt.Println("  1. 嵌入指针类型时，指针可能为 nil，调用方法会 panic")
	fmt.Println("  2. 嵌入值类型时，总是有零值，不会因 nil 而 panic")
	fmt.Println("  3. 如果必须嵌入指针，确保在构造函数中初始化")
	fmt.Println("  4. 优先嵌入值类型，除非需要多态或共享底层数据")
}
