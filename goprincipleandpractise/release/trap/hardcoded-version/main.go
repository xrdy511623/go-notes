package hardcodedversion

/*
陷阱：硬编码版本号

问题说明：
  把版本号写成常量（const version = "v1.2.3"），需要手动维护：

  1. 开发者忘记更新版本号 → 新版本显示旧版本号
  2. 版本号在多个地方定义 → 不一致
  3. 热修复时忘记改版本 → 无法区分修复前后
  4. CI/CD 中需要额外步骤修改源码

  const version = "v1.2.3"  // 上次发版时设的
  // ... 修了 10 个 bug ...
  // 忘记改版本号了
  // 线上运行的 v1.2.4 显示 "v1.2.3"

正确做法：
  用 var（不是 const）定义版本变量，编译时通过 ldflags 注入：

  var version = "dev"  // 默认值，开发时使用

  编译：go build -ldflags="-X 'main.version=v1.2.4'"

  好处：
  - 版本号由 Git tag 自动决定
  - 无需修改源码
  - CI/CD 中自动注入
  - 不可能忘记更新
*/

import "fmt"

// ❌ 错误做法：硬编码版本常量
const hardcodedVersion = "v1.2.3" // 最后一次手动更新是 3 个月前

// ✅ 正确做法：var 变量，由 ldflags 注入
var (
	injectedVersion   = "dev"
	injectedCommit    = "none"
	injectedBuildTime = "unknown"
)

// GetHardcodedVersion 返回硬编码的版本（可能已过时）
func GetHardcodedVersion() string {
	return hardcodedVersion
}

// GetInjectedVersion 返回 ldflags 注入的版本
func GetInjectedVersion() string {
	return fmt.Sprintf("%s (commit: %s, built: %s)",
		injectedVersion, injectedCommit, injectedBuildTime)
}

// VersionMismatchScenarios 列出硬编码版本导致的问题场景
func VersionMismatchScenarios() []string {
	return []string{
		"发版后忘记修改版本常量 → 新版显示旧版本号",
		"多处定义版本号 → 不同地方显示不同版本",
		"hotfix 后没改版本 → 无法区分修复前后的二进制",
		"CI/CD 需要额外步骤 sed 替换源码中的版本号",
		"const 不能被 ldflags 覆盖 → 必须用 var",
	}
}

// PrintProblem 打印问题说明
func PrintProblem() {
	fmt.Println("=== 硬编码版本号的问题 ===")
	fmt.Println()
	fmt.Printf("硬编码版本：%s（可能已过时 3 个月）\n", GetHardcodedVersion())
	fmt.Printf("注入版本：  %s\n", GetInjectedVersion())
	fmt.Println()
	fmt.Println("问题场景：")
	for i, s := range VersionMismatchScenarios() {
		fmt.Printf("  %d. %s\n", i+1, s)
	}
}
