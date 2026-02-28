package overusegenerics

/*
陷阱：过度使用泛型

问题说明：
  泛型的目的是消除重复代码，而非展示技巧。以下几种情况属于过度使用：

  1. 只有一种具体类型时使用泛型 → 增加复杂度，毫无收益
  2. 不同类型有不同逻辑时用泛型 → 应该用接口多态
  3. 为简单函数套泛型 → 一行代码变三行，可读性下降

  过度泛型化的代码特征：
  - 函数签名比函数体还复杂
  - 类型参数只被使用了一次
  - 读者需要追踪类型参数才能理解代码

正确做法：
  先写具体类型的代码，当出现第三次重复时再提取泛型。
  "Rule of Three"——两次重复是巧合，三次才是模式。
*/

import "fmt"

// ======= 错误示例 =======

// ❌ 只有一种类型时用泛型：毫无意义的泛型包装
func BadWrap[T any](value T) *T {
	return &value
}

// ✅ 直接用具体类型即可
func WrapString(value string) *string {
	return &value
}

// ❌ 泛型签名比函数体还复杂
func BadGetFirst[T any](s []T) (T, bool) {
	var zero T
	if len(s) == 0 {
		return zero, false
	}
	return s[0], true
}

// 如果你只在 []string 上用，直接写具体类型：
func GetFirstString(s []string) (string, bool) {
	if len(s) == 0 {
		return "", false
	}
	return s[0], true
}

// ======= 正确使用泛型的场景 =======

// ✅ 当确实有多种类型需要相同逻辑时，泛型才有价值
// 比如：同时在 []int, []string, []User 上使用 Filter
func Filter[T any](s []T, pred func(T) bool) []T {
	var result []T
	for _, v := range s {
		if pred(v) {
			result = append(result, v)
		}
	}
	return result
}

// PrintProblem 打印过度泛型化的问题
func PrintProblem() {
	fmt.Println("=== 过度使用泛型 ===")
	fmt.Println()
	fmt.Println("判断标准：")
	fmt.Println("  1. 类型参数是否被多种类型实际使用？")
	fmt.Println("     → 只有一种 → 不需要泛型")
	fmt.Println("  2. 泛型是否让代码更简单？")
	fmt.Println("     → 更复杂 → 不需要泛型")
	fmt.Println("  3. 是否至少有 3 处重复？")
	fmt.Println("     → 少于 3 处 → 暂时不需要泛型")
}
