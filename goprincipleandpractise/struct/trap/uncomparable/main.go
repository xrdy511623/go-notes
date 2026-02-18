package main

import (
	"fmt"
	"reflect"
)

/*
陷阱：包含不可比较字段的 struct 在运行时比较会 panic

运行：go run .

预期行为：
  Go 的 struct 是否可比较取决于所有字段是否可比较：
  - 可比较类型: int, string, bool, 指针, channel, 数组(元素可比较), struct(字段可比较)
  - 不可比较类型: slice, map, func

  包含不可比较字段的 struct：
  - 编译期: 直接用 == 比较会编译报错
  - 运行期: 通过 interface{} 比较会触发 panic

  正确做法: 使用 reflect.DeepEqual 或自定义 Equal 方法。
*/

// Comparable 所有字段都可比较
type Comparable struct {
	Name string
	Age  int
}

// Uncomparable 包含 slice 字段，不可比较
type Uncomparable struct {
	Name string
	Tags []string // slice 不可比较
}

// UncomparableMap 包含 map 字段，不可比较
type UncomparableMap struct {
	Labels map[string]string
	Name   string
}

func main() {
	fmt.Println("=== 演示1: 可比较的 struct ===")
	a := Comparable{Name: "Alice", Age: 30}
	b := Comparable{Name: "Alice", Age: 30}
	c := Comparable{Name: "Bob", Age: 25}
	fmt.Printf("  a == b: %v\n", a == b) // true
	fmt.Printf("  a == c: %v\n", a == c) // false
	fmt.Println("  可比较的 struct 可以直接用 == 比较，也可以作为 map key")

	fmt.Println("\n=== 演示2: 不可比较的 struct — 编译期报错 ===")
	fmt.Println("  // 以下代码无法编译:")
	fmt.Println("  // u1 := Uncomparable{Name: \"Alice\", Tags: []string{\"go\"}}")
	fmt.Println("  // u2 := Uncomparable{Name: \"Alice\", Tags: []string{\"go\"}}")
	fmt.Println("  // fmt.Println(u1 == u2)  // 编译错误: invalid operation")

	fmt.Println("\n=== 演示3: 通过 interface{} 比较 — 运行时 panic ===")
	u1 := Uncomparable{Name: "Alice", Tags: []string{"go"}}
	u2 := Uncomparable{Name: "Alice", Tags: []string{"go"}}

	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("  panic recovered: %v\n", r)
			}
		}()
		// 转为 interface{} 后，编译器不再检查可比较性
		// 运行时发现包含 slice，触发 panic
		var i1, i2 any = u1, u2
		_ = i1 == i2 // panic!
	}()

	fmt.Println("\n=== 演示4: 正确做法 — reflect.DeepEqual ===")
	fmt.Printf("  reflect.DeepEqual(u1, u2): %v\n", reflect.DeepEqual(u1, u2))

	u3 := Uncomparable{Name: "Alice", Tags: []string{"rust"}}
	fmt.Printf("  reflect.DeepEqual(u1, u3): %v\n", reflect.DeepEqual(u1, u3))

	fmt.Println("\n=== 演示5: 利用不可比较性阻止 struct 比较 ===")
	fmt.Println("  // 在 struct 中嵌入 [0]func() 或 _ [0]func() 可以使其不可比较")
	fmt.Println("  // 这是标准库中的技巧（如 sync.noCopy 使用 sync.Mutex 的方式）")
	type DoNotCompare struct {
		_    [0]func() // 零大小，不占内存，但使 struct 不可比较
		Name string
	}
	fmt.Printf("  unsafe.Sizeof(DoNotCompare{}) 中 _ 字段占 0 字节\n")

	fmt.Println("\n总结:")
	fmt.Println("  1. 包含 slice/map/func 的 struct 不可比较")
	fmt.Println("  2. 不可比较的 struct 通过 interface{} 比较会 runtime panic")
	fmt.Println("  3. 使用 reflect.DeepEqual 或自定义 Equal 方法比较")
	fmt.Println("  4. 嵌入 [0]func() 可以故意使 struct 不可比较")
}
