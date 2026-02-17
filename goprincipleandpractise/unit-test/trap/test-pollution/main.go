// Package main 演示共享状态导致的测试污染
//
// 当多个测试读写同一个包级变量（全局状态）时，
// 测试结果会依赖执行顺序，导致：
// 1. 单独运行通过，一起运行失败
// 2. 加 -count=N 出现间歇性失败
// 3. 加 -shuffle 随机失败
//
// 运行方式：go run ./goprincipleandpractise/unit-test/trap/test-pollution/
package main

import "fmt"

// ❌ 全局可变状态：测试污染的根源
var globalCounter int
var globalConfig = map[string]string{
	"mode": "production",
}

// 模拟测试 A：修改全局状态
func testA() {
	globalCounter++
	globalConfig["mode"] = "testing"
	fmt.Printf("  TestA: counter=%d, mode=%s\n", globalCounter, globalConfig["mode"])
}

// 模拟测试 B：依赖全局状态的初始值
func testB() {
	fmt.Printf("  TestB: counter=%d, mode=%s\n", globalCounter, globalConfig["mode"])
	if globalCounter != 0 {
		fmt.Println("  ❌ TestB FAIL: counter should be 0 (polluted by TestA)")
	} else {
		fmt.Println("  ✅ TestB PASS")
	}
	if globalConfig["mode"] != "production" {
		fmt.Println("  ❌ TestB FAIL: mode should be 'production' (polluted by TestA)")
	} else {
		fmt.Println("  ✅ TestB PASS")
	}
}

func main() {
	fmt.Println("=== 测试状态污染演示 ===")
	fmt.Println()

	// 场景 1：先 A 后 B（B 失败）
	fmt.Println("--- 执行顺序: A → B ---")
	globalCounter = 0
	globalConfig["mode"] = "production"
	testA()
	testB()

	fmt.Println()

	// 场景 2：先 B 后 A（B 通过）
	fmt.Println("--- 执行顺序: B → A ---")
	globalCounter = 0
	globalConfig["mode"] = "production"
	testB()
	testA()

	fmt.Println()
	fmt.Println("--- 解决方案 ---")
	fmt.Println("  1. 避免包级可变状态，通过参数或构造函数传入依赖")
	fmt.Println("  2. 每个测试在 setup 阶段重置状态：")
	fmt.Println("     t.Cleanup(func() { globalCounter = 0 })")
	fmt.Println("  3. 使用 t.Setenv() 代替 os.Setenv()（自动恢复）")
	fmt.Println("  4. 用 -shuffle 发现隐藏的顺序依赖：")
	fmt.Println("     go test -shuffle=on ./...")
}
