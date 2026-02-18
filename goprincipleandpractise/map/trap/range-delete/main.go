package main

import (
	"fmt"
)

/*
Go 语言规范明确保证：range 遍历 map 的过程中 delete 是安全的。

	"If a map entry that has not yet been reached is removed during iteration,
	 the corresponding iteration value will not be produced.
	 If a map entry is created during iteration, that entry may be produced
	 during the iteration or may be skipped."
	 — The Go Programming Language Specification

注意：
  - range 中 delete 安全（包括删除当前 key 和其他 key）
  - range 中 insert 行为不确定——新插入的 key 可能出现也可能不出现在后续迭代中
*/

func main() {
	// 示例 1：range 中删除偶数 key
	m := map[int]string{
		1: "a",
		2: "b",
		3: "c",
		4: "d",
		5: "e",
		6: "f",
	}
	fmt.Println("删除前:", m)

	for k := range m {
		if k%2 == 0 {
			delete(m, k) // 安全：删除当前 key
		}
	}
	fmt.Println("删除偶数 key 后:", m)

	// 示例 2：range 中删除满足条件的其他 key（也安全）
	m2 := map[string]int{
		"keep_a":   1,
		"keep_b":   2,
		"remove_c": 3,
		"remove_d": 4,
	}
	fmt.Println("\n删除前:", m2)

	for k := range m2 {
		if len(k) > 6 { // 删除 "remove_" 前缀的 key
			delete(m2, k)
		}
	}
	fmt.Println("删除 remove_ 前缀后:", m2)
}
