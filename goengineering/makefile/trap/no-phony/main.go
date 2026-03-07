package nophony

/*
陷阱：Makefile 中不声明 .PHONY

问题说明：
  Make 原始设计是面向文件的构建系统。当你写了一个 target 名叫 "build"，
  Make 会检查当前目录下是否有一个叫 "build" 的文件：
  - 如果有，且该文件比依赖更新 → Make 认为"目标已达成"，跳过执行
  - 如果没有 → 正常执行 target

  在 Go 项目中，如果你不小心创建了一个叫 build、test、clean 的文件或目录，
  对应的 Makefile target 就会失效——make 会输出 "make: 'build' is up to date."
  然后什么都不做。

  这种 bug 极其隐蔽：
  1. 本地开发时偶然创建同名文件 → CI 正常，本地 make 失效
  2. 某些工具生成了 build/ 目录 → make build 静默跳过
  3. 脚本写了 touch test → make test 不再执行

正确做法：
  声明 .PHONY，告诉 Make 这些 target 不是文件名：

  .PHONY: build test clean lint fmt vet run

  或者集中声明：

  .PHONY: all build test lint fmt vet clean run cover help

错误的 Makefile 示例：

  # 没有 .PHONY 声明！
  build:
  	go build -o bin/myapp .

  test:
  	go test ./...

  clean:
  	rm -rf bin/

  # 当目录下有 build 文件时：
  # $ make build
  # make: 'build' is up to date.   ← 什么都没做！

正确的 Makefile 示例：

  .PHONY: build test clean

  build:
  	go build -o bin/myapp .

  test:
  	go test ./...

  clean:
  	rm -rf bin/
*/

import "fmt"

// DemonstratePhonyProblem 演示 .PHONY 缺失导致的问题
// 当目录下存在与 target 同名的文件时，Make 会跳过该 target
func DemonstratePhonyProblem() {
	fmt.Println("=== .PHONY 缺失问题演示 ===")
	fmt.Println()
	fmt.Println("场景：项目目录下有一个叫 'build' 的文件")
	fmt.Println()
	fmt.Println("没有 .PHONY 时：")
	fmt.Println("  $ make build")
	fmt.Println("  make: 'build' is up to date.")
	fmt.Println("  （什么都没做！编译被跳过了！）")
	fmt.Println()
	fmt.Println("有 .PHONY 时：")
	fmt.Println("  $ make build")
	fmt.Println("  Building myapp v1.0.0...")
	fmt.Println("  （正常执行编译）")
	fmt.Println()
	fmt.Println("常见触发场景：")
	fmt.Println("  1. build/ 目录（Go 项目常见）")
	fmt.Println("  2. test 文件（测试数据）")
	fmt.Println("  3. clean 脚本")
	fmt.Println("  4. CI 流水线中缓存了上次的产物目录")
}

// PhonyTargets 列出应该声明为 .PHONY 的常见 target
func PhonyTargets() []string {
	return []string{
		"all",      // 默认目标
		"build",    // 编译
		"test",     // 测试
		"lint",     // 代码检查
		"fmt",      // 格式化
		"vet",      // 静态分析
		"clean",    // 清理
		"run",      // 运行
		"cover",    // 覆盖率
		"help",     // 帮助
		"generate", // 代码生成
		"proto",    // protobuf 编译
		"mock",     // mock 生成
		"docker",   // Docker 构建
		"deploy",   // 部署
		"release",  // 发布
	}
}
