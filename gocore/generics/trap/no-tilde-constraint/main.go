package notildeconstraint

/*
陷阱：约束中不使用 ~ 导致自定义类型被拒绝

问题说明：
  Go 约束中 int 和 ~int 的区别：
  - int：只接受 int 类型本身
  - ~int：接受底层类型为 int 的所有类型（包括 type MyInt int）

  实际工程中，自定义类型非常常见：
    type UserID int64
    type Money int64
    type Score float64

  如果约束中不使用 ~，这些自定义类型全部无法使用你的泛型函数，
  导致调用者被迫做类型转换，体验极差。

正确做法：
  除非你有明确理由只接受原始类型，否则约束中应该始终使用 ~T。
  更好的做法是直接使用 cmp.Ordered，它已经包含了 ~ 前缀。
*/

import "fmt"

// ❌ 错误：约束不使用 ~
type StrictOrdered interface {
	int | int64 | float64 | string
}

// ✅ 正确：约束使用 ~
type FlexOrdered interface {
	~int | ~int64 | ~float64 | ~string
}

// StrictMax 只接受原始类型
func StrictMax[T StrictOrdered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// FlexMax 接受原始类型和自定义类型
func FlexMax[T FlexOrdered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// 自定义类型
type UserID int64
type Money int64

// DemonstrateProblem 演示 ~ 的重要性
func DemonstrateProblem() {
	fmt.Println("=== 约束中不使用 ~ 的问题 ===")
	fmt.Println()

	// 原始类型：两种约束都可以
	fmt.Println("原始类型 int:")
	fmt.Printf("  StrictMax(1, 2) = %d\n", StrictMax(1, 2))
	fmt.Printf("  FlexMax(1, 2) = %d\n", FlexMax(1, 2))

	// 自定义类型：只有 FlexMax 可以
	fmt.Println()
	fmt.Println("自定义类型 UserID:")
	var a, b UserID = 100, 200
	// StrictMax(a, b) → 编译错误：UserID does not satisfy StrictOrdered
	fmt.Printf("  FlexMax(%d, %d) = %d\n", a, b, FlexMax(a, b))

	fmt.Println()
	fmt.Println("结论：约束中始终使用 ~T，或直接使用 cmp.Ordered")
}
