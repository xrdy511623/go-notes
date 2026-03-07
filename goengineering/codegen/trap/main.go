package main

import "fmt"

func main() {
	trapForgotGenerate()
	trapManualEditGenerated()
	trapDirectiveSpacing()
	trapOutOfSyncEnum()
}

// ============================================================
// 陷阱1：忘记运行go generate
// 修改了枚举值但没重新生成，String()返回 "Type(N)" 而非名称
// ============================================================

func trapForgotGenerate() {
	fmt.Println("=== 陷阱1：忘记运行go generate ===")
	fmt.Println("场景：在Color枚举中新增了Purple，但忘了运行go generate")
	fmt.Println("后果：Purple.String() 返回 \"Color(4)\" 而非 \"Purple\"")
	fmt.Println("解决：CI中加 generate-check 步骤：")
	fmt.Println("  go generate ./...")
	fmt.Println("  git diff --exit-code || (echo '生成文件未更新' && exit 1)")
	fmt.Println()
}

// ============================================================
// 陷阱2：手动修改生成文件
// ============================================================

func trapManualEditGenerated() {
	fmt.Println("=== 陷阱2：手动修改生成文件 ===")
	fmt.Println("场景：在 color_string.go 中手动修改了 \"Red\" → \"红色\"")
	fmt.Println("后果：下次 go generate 会覆盖你的修改")
	fmt.Println("解决：")
	fmt.Println("  - 修改源定义（用 -linecomment 自定义字符串）")
	fmt.Println("  - 生成文件的第一行 'Code generated ... DO NOT EDIT' 就是在提醒你")
	fmt.Println()
}

// ============================================================
// 陷阱3：generate指令格式错误
// ============================================================

type Ignored int

// 下面这些指令格式有问题（故意用注释说明）

// // go:generate stringer -type=Ignored    ← ❌ //和go:之间有空格，不会被执行
// /*go:generate stringer -type=Ignored*/  ← ❌ 块注释不支持
// //go:Generate stringer -type=Ignored    ← ❌ 大小写敏感，必须是generate

func trapDirectiveSpacing() {
	fmt.Println("=== 陷阱3：generate指令格式错误 ===")
	fmt.Println("常见错误：")
	fmt.Println("  // go:generate ...  ← ❌ //后有空格")
	fmt.Println("  //go:Generate ...   ← ❌ 大小写错误")
	fmt.Println("  /*go:generate ...*/  ← ❌ 块注释不支持")
	fmt.Println("正确格式：")
	fmt.Println("  //go:generate ...   ← ✅ //紧接go:generate")
	fmt.Println()
	fmt.Println("调试技巧：go generate -n ./... 可以看到哪些指令会被执行")
	fmt.Println()
}

// ============================================================
// 陷阱4：枚举值与生成代码不同步
// ============================================================

func trapOutOfSyncEnum() {
	fmt.Println("=== 陷阱4：枚举值顺序变更导致不同步 ===")
	fmt.Println("场景：在已有枚举的中间插入新值")
	fmt.Println()
	fmt.Println("  const (")
	fmt.Println("      Red   = iota  // 0")
	fmt.Println("      Green         // 1 ← 原来是1")
	fmt.Println("      Blue          // 2 ← 原来是2")
	fmt.Println("  )")
	fmt.Println()
	fmt.Println("  插入Yellow后：")
	fmt.Println("  const (")
	fmt.Println("      Red    = iota  // 0")
	fmt.Println("      Yellow         // 1 ← 新增")
	fmt.Println("      Green          // 2 ← 变了！原来是1")
	fmt.Println("      Blue           // 3 ← 变了！原来是2")
	fmt.Println("  )")
	fmt.Println()
	fmt.Println("后果：数据库中存储的1(原Green)现在变成了Yellow")
	fmt.Println("解决：")
	fmt.Println("  1. 新增枚举值放在末尾，不要插入中间")
	fmt.Println("  2. 或者显式指定数值：Green = 1, Blue = 2, Yellow = 3")
	fmt.Println()
}
