// Package main 演示 govulncheck 调用图分析的核心概念。
//
// 核心教训：导入了含漏洞的包 ≠ 你的代码受影响。
// govulncheck 通过调用图分析，只在代码实际调用漏洞函数时才报警。
//
// 运行方式：
//
//	govulncheck ./...
//
// 场景说明：
// 假设依赖模块 example.com/lib 的 v1.2.0 存在已知漏洞 GO-2024-XXXX，
// 漏洞位于 lib.VulnerableFunc() 函数中。
//
// 场景 A（安全）：导入了 lib 包，但只调用了 lib.SafeFunc()。
// govulncheck 分析调用图后发现 VulnerableFunc 不可达 → 不报警。
//
// 场景 B（受影响）：代码调用了 lib.VulnerableFunc()。
// govulncheck 追踪到 main → processData → lib.VulnerableFunc → 漏洞代码 → 报警。
//
// 对比 go list -m all：
// go list -m all 只看到 "example.com/lib v1.2.0" 在依赖列表中，
// 无法区分场景 A 和 B，两种情况都会标记为"有漏洞"（误报场景 A）。
package main

import "fmt"

// 以下用伪代码模拟两种场景，因为真实复现需要引入已知含漏洞的特定版本依赖。

// --- 场景 A：安全（漏洞函数不可达）---

func scenarioA() {
	// import "example.com/lib"
	// result := lib.SafeFunc(data)  // 只调用了安全函数
	// lib.VulnerableFunc 从未被调用
	//
	// govulncheck 结果：No vulnerabilities found.
	// go list -m all 结果：example.com/lib v1.2.0（标记为有漏洞 — 误报！）
	fmt.Println("场景 A：导入了含漏洞的包，但未调用漏洞函数 → 安全")
}

// --- 场景 B：受影响（漏洞函数可达）---

func scenarioB() {
	// import "example.com/lib"
	// result := lib.VulnerableFunc(data)  // 直接调用了漏洞函数！
	//
	// govulncheck 输出：
	//   Vulnerability #1: GO-2024-XXXX
	//     Found in: example.com/lib@v1.2.0
	//     Fixed in: example.com/lib@v1.3.0
	//     Example trace:
	//       main.go → scenarioB → lib.VulnerableFunc
	//
	// 修复：go get example.com/lib@v1.3.0
	fmt.Println("场景 B：调用了漏洞函数 → govulncheck 报警，需要升级依赖")
}

func main() {
	fmt.Println("=== govulncheck 调用图分析演示 ===")
	fmt.Println()
	scenarioA()
	fmt.Println()
	scenarioB()
	fmt.Println()
	fmt.Println("关键区别：")
	fmt.Println("  go list -m all  → 两种场景都标记为'有漏洞'（场景 A 是误报）")
	fmt.Println("  govulncheck     → 只有场景 B 报警（调用图可达）")
}
