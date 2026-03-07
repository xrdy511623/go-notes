package main

import "fmt"

/*
nil map 与 empty map 的行为差异：

nil map（仅声明未初始化）：
  - 读操作安全，返回 value 类型的零值
  - len() 返回 0
  - range 不执行循环体
  - 写操作 panic: assignment to entry in nil map

empty map（通过 make 或字面量 {} 创建）：
  - 读写均安全

最佳实践：始终通过 make(map[K]V) 或字面量 map[K]V{} 初始化 map。
*/

func main() {
	// ---- nil map ----
	var m map[string]int
	fmt.Println("== nil map ==")
	fmt.Println("m == nil:", m == nil)   // true
	fmt.Println("len(m):", len(m))       // 0
	fmt.Println("m[\"key\"]:", m["key"]) // 0（零值，不 panic）

	v, ok := m["missing"]
	fmt.Printf("v=%d, ok=%v\n", v, ok) // v=0, ok=false

	// range nil map 安全，不执行循环体
	for k, v := range m {
		fmt.Println(k, v) // 不会执行
	}

	// 写入 nil map 会 panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("写入 nil map panic:", r)
			}
		}()
		m["key"] = 1 // panic: assignment to entry in nil map
	}()

	// delete nil map 安全（Go 1.0 之后）
	delete(m, "key") // 不 panic
	fmt.Println("delete nil map: safe")

	// ---- empty map ----
	fmt.Println("\n== empty map ==")
	m1 := map[string]int{}
	m2 := make(map[string]int)

	fmt.Println("m1 == nil:", m1 == nil) // false
	fmt.Println("m2 == nil:", m2 == nil) // false

	m1["key"] = 1 // 安全
	m2["key"] = 2 // 安全
	fmt.Println("m1[\"key\"]:", m1["key"])
	fmt.Println("m2[\"key\"]:", m2["key"])
}
